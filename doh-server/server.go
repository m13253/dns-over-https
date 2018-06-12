/*
   DNS-over-HTTPS
   Copyright (C) 2017-2018 Star Brilliant <m13253@hotmail.com>

   Permission is hereby granted, free of charge, to any person obtaining a
   copy of this software and associated documentation files (the "Software"),
   to deal in the Software without restriction, including without limitation
   the rights to use, copy, modify, merge, publish, distribute, sublicense,
   and/or sell copies of the Software, and to permit persons to whom the
   Software is furnished to do so, subject to the following conditions:

   The above copyright notice and this permission notice shall be included in
   all copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
   FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
   DEALINGS IN THE SOFTWARE.
*/

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/m13253/dns-over-https/json-dns"
	"github.com/miekg/dns"
)

type Server struct {
	conf      *config
	udpClient *dns.Client
	tcpClient *dns.Client
	servemux  *http.ServeMux
}

type DNSRequest struct {
	request         *dns.Msg
	response        *dns.Msg
	currentUpstream string
	isTailored      bool
	errcode         int
	errtext         string
}

func NewServer(conf *config) (s *Server) {
	s = &Server{
		conf: conf,
		udpClient: &dns.Client{
			Net:     "udp",
			UDPSize: dns.DefaultMsgSize,
			Timeout: time.Duration(conf.Timeout) * time.Second,
		},
		tcpClient: &dns.Client{
			Net:     "tcp",
			Timeout: time.Duration(conf.Timeout) * time.Second,
		},
		servemux: http.NewServeMux(),
	}
	s.servemux.HandleFunc(conf.Path, s.handlerFunc)
	return
}

func (s *Server) Start() error {
	servemux := http.Handler(s.servemux)
	if s.conf.Verbose {
		servemux = handlers.CombinedLoggingHandler(os.Stdout, servemux)
	}
	results := make(chan error, len(s.conf.Listen))
	for _, addr := range s.conf.Listen {
		go func(addr string) {
			var err error
			if s.conf.Cert != "" || s.conf.Key != "" {
				err = http.ListenAndServeTLS(addr, s.conf.Cert, s.conf.Key, servemux)
			} else {
				err = http.ListenAndServe(addr, servemux)
			}
			if err != nil {
				log.Println(err)
			}
			results <- err
		}(addr)
	}
	// wait for all handlers
	for i := 0; i < cap(results); i++ {
		err := <-results
		if err != nil {
			return err
		}
	}
	close(results)
	return nil
}

func (s *Server) handlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", USER_AGENT)
	w.Header().Set("X-Powered-By", USER_AGENT)

	if r.Form == nil {
		const maxMemory = 32 << 20 // 32 MB
		r.ParseMultipartForm(maxMemory)
	}
	contentType := r.Header.Get("Content-Type")
	if ct := r.FormValue("ct"); ct != "" {
		contentType = ct
	}
	if contentType == "" {
		// Guess request Content-Type based on other parameters
		if r.FormValue("name") != "" {
			contentType = "application/dns-json"
		} else if r.FormValue("dns") != "" {
			contentType = "application/dns-message"
		}
	}
	var responseType string
	for _, responseCandidate := range strings.Split(r.Header.Get("Accept"), ",") {
		responseCandidate = strings.SplitN(responseCandidate, ";", 2)[0]
		if responseCandidate == "application/json" {
			responseType = "application/json"
			break
		} else if responseCandidate == "application/dns-udpwireformat" {
			responseType = "application/dns-message"
			break
		} else if responseCandidate == "application/dns-message" {
			responseType = "application/dns-message"
			break
		}
	}
	if responseType == "" {
		// Guess response Content-Type based on request Content-Type
		if contentType == "application/dns-json" {
			responseType = "application/json"
		} else if contentType == "application/dns-message" {
			responseType = "application/dns-message"
		} else if contentType == "application/dns-udpwireformat" {
			responseType = "application/dns-message"
		}
	}

	var req *DNSRequest
	if contentType == "application/dns-json" {
		req = s.parseRequestGoogle(w, r)
	} else if contentType == "application/dns-message" {
		req = s.parseRequestIETF(w, r)
	} else if contentType == "application/dns-udpwireformat" {
		req = s.parseRequestIETF(w, r)
	} else {
		jsonDNS.FormatError(w, fmt.Sprintf("Invalid argument value: \"ct\" = %q", contentType), 415)
		return
	}
	if req.errcode != 0 {
		jsonDNS.FormatError(w, req.errtext, req.errcode)
		return
	}

	var err error
	req, err = s.doDNSQuery(req)
	if err != nil {
		jsonDNS.FormatError(w, fmt.Sprintf("DNS query failure (%s)", err.Error()), 503)
		return
	}

	if responseType == "application/json" {
		s.generateResponseGoogle(w, r, req)
	} else if responseType == "application/dns-message" {
		s.generateResponseIETF(w, r, req)
	} else {
		panic("Unknown response Content-Type")
	}
}

func (s *Server) findClientIP(r *http.Request) net.IP {
	XForwardedFor := r.Header.Get("X-Forwarded-For")
	if XForwardedFor != "" {
		for _, addr := range strings.Split(XForwardedFor, ",") {
			addr = strings.TrimSpace(addr)
			ip := net.ParseIP(addr)
			if jsonDNS.IsGlobalIP(ip) {
				return ip
			}
		}
	}
	XRealIP := r.Header.Get("X-Real-IP")
	if XRealIP != "" {
		addr := strings.TrimSpace(XRealIP)
		ip := net.ParseIP(addr)
		if jsonDNS.IsGlobalIP(ip) {
			return ip
		}
	}
	remoteAddr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
	if err != nil {
		return nil
	}
	if ip := remoteAddr.IP; jsonDNS.IsGlobalIP(ip) {
		return ip
	}
	return nil
}

func (s *Server) doDNSQuery(req *DNSRequest) (resp *DNSRequest, err error) {
	numServers := len(s.conf.Upstream)
	for i := uint(0); i < s.conf.Tries; i++ {
		req.currentUpstream = s.conf.Upstream[rand.Intn(numServers)]
		if !s.conf.TCPOnly {
			req.response, _, err = s.udpClient.Exchange(req.request, req.currentUpstream)
			if err == dns.ErrTruncated {
				log.Println(err)
				req.response, _, err = s.tcpClient.Exchange(req.request, req.currentUpstream)
			}
		} else {
			req.response, _, err = s.tcpClient.Exchange(req.request, req.currentUpstream)
		}
		if err == nil {
			return req, nil
		}
		log.Printf("DNS error from upstream %s: %s\n", req.currentUpstream, err.Error())
	}
	return req, err
}
