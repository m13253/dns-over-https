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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	jsondns "github.com/m13253/dns-over-https/v2/json-dns"
)

func (s *Server) parseRequestIETF(ctx context.Context, w http.ResponseWriter, r *http.Request) *DNSRequest {
	requestBase64 := r.FormValue("dns")
	requestBinary, err := base64.RawURLEncoding.DecodeString(requestBase64)
	if err != nil {
		return &DNSRequest{
			errcode: 400,
			errtext: fmt.Sprintf("Invalid argument value: \"dns\" = %q", requestBase64),
		}
	}
	if len(requestBinary) == 0 && (r.Header.Get("Content-Type") == "application/dns-message" || r.Header.Get("Content-Type") == "application/dns-udpwireformat") {
		requestBinary, err = io.ReadAll(r.Body)
		if err != nil {
			return &DNSRequest{
				errcode: 400,
				errtext: fmt.Sprintf("Failed to read request body (%s)", err.Error()),
			}
		}
	}
	if len(requestBinary) == 0 {
		return &DNSRequest{
			errcode: 400,
			errtext: fmt.Sprintf("Invalid argument value: \"dns\""),
		}
	}

	if s.patchDNSCryptProxyReqID(w, r, requestBinary) {
		return &DNSRequest{
			errcode: 444,
		}
	}

	msg := new(dns.Msg)
	err = msg.Unpack(requestBinary)
	if err != nil {
		return &DNSRequest{
			errcode: 400,
			errtext: fmt.Sprintf("DNS packet parse failure (%s)", err.Error()),
		}
	}

	if s.conf.Verbose && len(msg.Question) > 0 {
		question := &msg.Question[0]
		questionName := question.Name
		questionClass := ""
		if qclass, ok := dns.ClassToString[question.Qclass]; ok {
			questionClass = qclass
		} else {
			questionClass = strconv.FormatUint(uint64(question.Qclass), 10)
		}
		questionType := ""
		if qtype, ok := dns.TypeToString[question.Qtype]; ok {
			questionType = qtype
		} else {
			questionType = strconv.FormatUint(uint64(question.Qtype), 10)
		}
		var clientip net.IP = nil
		if s.conf.LogGuessedIP {
			clientip = s.findClientIP(r)
		}
		if clientip != nil {
			fmt.Printf("%s - - [%s] \"%s %s %s\"\n", clientip, time.Now().Format("02/Jan/2006:15:04:05 -0700"), questionName, questionClass, questionType)
		} else {
			fmt.Printf("%s - - [%s] \"%s %s %s\"\n", r.RemoteAddr, time.Now().Format("02/Jan/2006:15:04:05 -0700"), questionName, questionClass, questionType)
		}
	}

	transactionID := msg.Id
	msg.Id = dns.Id()
	opt := msg.IsEdns0()
	if opt == nil {
		opt = new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(dns.DefaultMsgSize)
		opt.SetDo(false)
		msg.Extra = append([]dns.RR{opt}, msg.Extra...)
	}
	var edns0Subnet *dns.EDNS0_SUBNET
	for _, option := range opt.Option {
		if option.Option() == dns.EDNS0SUBNET {
			edns0Subnet = option.(*dns.EDNS0_SUBNET)
			break
		}
	}
	isTailored := edns0Subnet == nil

	if edns0Subnet == nil {
		ednsClientFamily := uint16(0)
		ednsClientAddress := s.findClientIP(r)
		ednsClientNetmask := uint8(255)
		if ednsClientAddress != nil {
			if ipv4 := ednsClientAddress.To4(); ipv4 != nil {
				ednsClientFamily = 1
				ednsClientAddress = ipv4
				if s.conf.ECSUsePreciseIP {
					ednsClientNetmask = 32
				} else {
					ednsClientNetmask = 24
					ednsClientAddress = ednsClientAddress.Mask(net.CIDRMask(24, 32))
				}
			} else {
				ednsClientFamily = 2
				if s.conf.ECSUsePreciseIP {
					ednsClientNetmask = 128
				} else {
					ednsClientNetmask = 56
					ednsClientAddress = ednsClientAddress.Mask(net.CIDRMask(56, 128))
				}
			}
			edns0Subnet = new(dns.EDNS0_SUBNET)
			edns0Subnet.Code = dns.EDNS0SUBNET
			edns0Subnet.Family = ednsClientFamily
			edns0Subnet.SourceNetmask = ednsClientNetmask
			edns0Subnet.SourceScope = 0
			edns0Subnet.Address = ednsClientAddress
			opt.Option = append(opt.Option, edns0Subnet)
		}
	}

	return &DNSRequest{
		request:       msg,
		transactionID: transactionID,
		isTailored:    isTailored,
	}
}

func (s *Server) generateResponseIETF(ctx context.Context, w http.ResponseWriter, r *http.Request, req *DNSRequest) {
	respJSON := jsondns.Marshal(req.response)
	req.response.Id = req.transactionID
	respBytes, err := req.response.Pack()
	if err != nil {
		log.Printf("DNS packet construct failure with upstream %s: %v\n", req.currentUpstream, err)
		jsondns.FormatError(w, fmt.Sprintf("DNS packet construct failure (%s)", err.Error()), 500)
		return
	}

	w.Header().Set("Content-Type", "application/dns-message")
	now := time.Now().UTC().Format(http.TimeFormat)
	w.Header().Set("Date", now)
	w.Header().Set("Last-Modified", now)
	w.Header().Set("Vary", "Accept")

	if respJSON.HaveTTL {
		if req.isTailored {
			w.Header().Set("Cache-Control", "private, max-age="+strconv.FormatUint(uint64(respJSON.LeastTTL), 10))
		} else {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.FormatUint(uint64(respJSON.LeastTTL), 10))
		}
		w.Header().Set("Expires", respJSON.EarliestExpires.Format(http.TimeFormat))
	}

	if respJSON.Status == dns.RcodeServerFailure {
		log.Printf("received server failure from upstream %s: %v\n", req.currentUpstream, req.response)
		w.WriteHeader(503)
	}
	_, err = w.Write(respBytes)
	if err != nil {
		log.Printf("failed to write to client: %v\n", err)
	}
}

// Workaround a bug causing DNSCrypt-Proxy to expect a response with TransactionID = 0xcafe.
func (s *Server) patchDNSCryptProxyReqID(w http.ResponseWriter, r *http.Request, requestBinary []byte) bool {
	if strings.Contains(r.UserAgent(), "dnscrypt-proxy") && bytes.Equal(requestBinary, []byte("\xca\xfe\x01\x00\x00\x01\x00\x00\x00\x00\x00\x01\x00\x00\x02\x00\x01\x00\x00\x29\x10\x00\x00\x00\x80\x00\x00\x00")) {
		if s.conf.Verbose {
			log.Println("DNSCrypt-Proxy detected. Patching response.")
		}
		w.Header().Set("Content-Type", "application/dns-message")
		w.Header().Set("Vary", "Accept, User-Agent")
		now := time.Now().UTC().Format(http.TimeFormat)
		w.Header().Set("Date", now)
		w.Write([]byte("\xca\xfe\x81\x05\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00\x02\x00\x01\x00\x00\x10\x00\x01\x00\x00\x00\x00\x00\xa8\xa7\r\nWorkaround a bug causing DNSCrypt-Proxy to expect a response with TransactionID = 0xcafe\r\nRefer to https://github.com/jedisct1/dnscrypt-proxy/issues/526 for details."))
		return true
	}
	return false
}
