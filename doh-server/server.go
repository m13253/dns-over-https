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
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	jsonDNS "github.com/m13253/dns-over-https/json-dns"
	"github.com/miekg/dns"
)

type Server struct {
	conf         *config
	udpClient    *dns.Client
	tcpClient    *dns.Client
	tcpClientTLS *dns.Client
	servemux     *http.ServeMux
}

type DNSRequest struct {
	request         *dns.Msg
	response        *dns.Msg
	transactionID   uint16
	currentUpstream string
	isTailored      bool
	errcode         int
	errtext         string
}

func NewServer(conf *config) (*Server, error) {
	timeout := time.Duration(conf.Timeout) * time.Second
	s := &Server{
		conf: conf,
		udpClient: &dns.Client{
			Net:     "udp",
			UDPSize: dns.DefaultMsgSize,
			Timeout: timeout,
		},
		tcpClient: &dns.Client{
			Net:     "tcp",
			Timeout: timeout,
		},
		tcpClientTLS: &dns.Client{
			Net:     "tcp-tls",
			Timeout: timeout,
		},
		servemux: http.NewServeMux(),
	}
	if conf.LocalAddr != "" {
		udpLocalAddr, err := net.ResolveUDPAddr("udp", conf.LocalAddr)
		if err != nil {
			return nil, err
		}
		tcpLocalAddr, err := net.ResolveTCPAddr("tcp", conf.LocalAddr)
		if err != nil {
			return nil, err
		}
		s.udpClient.Dialer = &net.Dialer{
			Timeout:   timeout,
			LocalAddr: udpLocalAddr,
		}
		s.tcpClient.Dialer = &net.Dialer{
			Timeout:   timeout,
			LocalAddr: tcpLocalAddr,
		}
		s.tcpClientTLS.Dialer = &net.Dialer{
			Timeout:   timeout,
			LocalAddr: tcpLocalAddr,
		}
	}
	s.servemux.HandleFunc(conf.Path, s.handlerFunc)
	return s, nil
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
	ctx := r.Context()

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.Header().Set("Server", USER_AGENT)
	w.Header().Set("X-Powered-By", USER_AGENT)

	if r.Method == "OPTIONS" {
		w.Header().Set("Content-Length", "0")
		return
	}

	if r.Form == nil {
		const maxMemory = 32 << 20 // 32 MB
		r.ParseMultipartForm(maxMemory)
	}

	for _, header := range s.conf.DebugHTTPHeaders {
		if value := r.Header.Get(header); value != "" {
			log.Printf("%s: %s\n", header, value)
		}
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
		req = s.parseRequestGoogle(ctx, w, r)
	} else if contentType == "application/dns-message" {
		req = s.parseRequestIETF(ctx, w, r)
	} else if contentType == "application/dns-udpwireformat" {
		req = s.parseRequestIETF(ctx, w, r)
	} else {
		jsonDNS.FormatError(w, fmt.Sprintf("Invalid argument value: \"ct\" = %q", contentType), 415)
		return
	}
	if req.errcode == 444 {
		return
	}
	if req.errcode != 0 {
		jsonDNS.FormatError(w, req.errtext, req.errcode)
		return
	}

	req = s.patchRootRD(req)

	var err error
	req, err = s.doDNSQuery(ctx, req)
	if err != nil {
		jsonDNS.FormatError(w, fmt.Sprintf("DNS query failure (%s)", err.Error()), 503)
		return
	}

	if responseType == "application/json" {
		s.generateResponseGoogle(ctx, w, r, req)
	} else if responseType == "application/dns-message" {
		s.generateResponseIETF(ctx, w, r, req)
	} else {
		panic("Unknown response Content-Type")
	}
}

func (s *Server) findClientIP(r *http.Request) net.IP {
	noEcs := r.URL.Query().Get("no_ecs")
	if strings.ToLower(noEcs) == "true" {
		return nil
	}

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

// Workaround a bug causing Unbound to refuse returning anything about the root
func (s *Server) patchRootRD(req *DNSRequest) *DNSRequest {
	for _, question := range req.request.Question {
		if question.Name == "." {
			req.request.RecursionDesired = true
		}
	}
	return req
}

// Return the position index for the question of qtype from a DNS msg, otherwise return -1
func (s *Server) indexQuestionType(msg *dns.Msg, qtype uint16) int {
	for i, question := range msg.Question {
		if question.Qtype == qtype {
			return i
		}
	}
	return -1
}

func (s *Server) doDNSQuery(ctx context.Context, req *DNSRequest) (resp *DNSRequest, err error) {
	// TODO(m13253): Make ctx work. Waiting for a patch for ExchangeContext from miekg/dns.
	numServers := len(s.conf.Upstream)
	for i := uint(0); i < s.conf.Tries; i++ {
		req.currentUpstream = s.conf.Upstream[rand.Intn(numServers)]

		upstream, t := addressAndType(req.currentUpstream)

		switch t {
		default:
			log.Printf("invalid DNS type %q in upstream %q", t, upstream)
			return nil, &configError{"invalid DNS type"}
		// Use DNS-over-TLS (DoT) if configured to do so
		case "tcp-tls":
			req.response, _, err = s.tcpClientTLS.Exchange(req.request, upstream)
		case "tcp", "udp":
			// Use TCP if always configured to or if the Query type dictates it (AXFR)
			if t == "tcp" || (s.indexQuestionType(req.request, dns.TypeAXFR) > -1) {
				req.response, _, err = s.tcpClient.Exchange(req.request, upstream)
			} else {
				req.response, _, err = s.udpClient.Exchange(req.request, upstream)
				if err == nil && req.response != nil && req.response.Truncated {
					log.Println(err)
					req.response, _, err = s.tcpClient.Exchange(req.request, upstream)
				}

				// Retry with TCP if this was an IXFR request and we only received an SOA
				if (s.indexQuestionType(req.request, dns.TypeIXFR) > -1) &&
					(len(req.response.Answer) == 1) &&
					(req.response.Answer[0].Header().Rrtype == dns.TypeSOA) {
					req.response, _, err = s.tcpClient.Exchange(req.request, upstream)
				}
			}
		}

		if err == nil {
			return req, nil
		}
		log.Printf("DNS error from upstream %s: %s\n", req.currentUpstream, err.Error())
	}
	return req, err
}
