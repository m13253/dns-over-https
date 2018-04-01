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

	"../json-dns"
	"github.com/gorilla/handlers"
	"github.com/miekg/dns"
)

type Server struct {
	conf      *config
	udpClient *dns.Client
	tcpClient *dns.Client
	servemux  *http.ServeMux
}

type DNSRequest struct {
	request    *dns.Msg
	response   *dns.Msg
	isTailored bool
	errcode    int
	errtext    string
}

func NewServer(conf *config) (s *Server) {
	s = &Server{
		conf: conf,
		udpClient: &dns.Client{
			Net:     "udp",
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
	if s.conf.Cert != "" || s.conf.Key != "" {
		return http.ListenAndServeTLS(s.conf.Listen, s.conf.Cert, s.conf.Key, servemux)
	}
	return http.ListenAndServe(s.conf.Listen, servemux)
}

func (s *Server) handlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "DNS-over-HTTPS/1.1 (+https://github.com/m13253/dns-over-https)")
	w.Header().Set("X-Powered-By", "DNS-over-HTTPS/1.1 (+https://github.com/m13253/dns-over-https)")

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
			contentType = "application/dns-udpwireformat"
		}
	}
	var responseType string
	for _, responseCandidate := range strings.Split(r.Header.Get("Accept"), ",") {
		responseCandidate = strings.ToLower(strings.SplitN(responseCandidate, ";", 2)[0])
		if responseCandidate == "application/json" {
			responseType = "application/json"
			break
		} else if responseCandidate == "application/dns-udpwireformat" {
			responseType = "application/dns-udpwireformat"
			break
		}
	}
	if responseType == "" {
		// Guess response Content-Type based on request Content-Type
		if contentType == "application/dns-json" {
			responseType = "application/json"
		} else if contentType == "application/dns-udpwireformat" {
			responseType = "application/dns-udpwireformat"
		}
	}

	var req *DNSRequest
	if contentType == "application/dns-json" {
		req = s.parseRequestGoogle(w, r)
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
	req.response, err = s.doDNSQuery(req.request)
	if err != nil {
		jsonDNS.FormatError(w, fmt.Sprintf("DNS query failure (%s)", err.Error()), 503)
		return
	}

	if responseType == "application/json" {
		s.generateResponseGoogle(w, r, req)
	} else if responseType == "application/dns-udpwireformat" {
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

func (s *Server) doDNSQuery(msg *dns.Msg) (resp *dns.Msg, err error) {
	numServers := len(s.conf.Upstream)
	for i := uint(0); i < s.conf.Tries; i++ {
		server := s.conf.Upstream[rand.Intn(numServers)]
		if !s.conf.TCPOnly {
			resp, _, err = s.udpClient.Exchange(msg, server)
			if err == dns.ErrTruncated {
				log.Println(err)
				resp, _, err = s.tcpClient.Exchange(msg, server)
			}
		} else {
			resp, _, err = s.tcpClient.Exchange(msg, server)
		}
		if err == nil {
			return
		}
		log.Println(err)
	}
	return
}
