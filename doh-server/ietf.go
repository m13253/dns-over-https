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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/m13253/dns-over-https/json-dns"
	"github.com/miekg/dns"
)

func (s *Server) parseRequestIETF(w http.ResponseWriter, r *http.Request) *DNSRequest {
	requestBase64 := r.FormValue("dns")
	requestBinary, err := base64.RawURLEncoding.DecodeString(requestBase64)
	if err != nil {
		return &DNSRequest{
			errcode: 400,
			errtext: fmt.Sprintf("Invalid argument value: \"dns\" = %q", requestBase64),
		}
	}
	if len(requestBinary) == 0 && (r.Header.Get("Content-Type") == "application/dns-message" || r.Header.Get("Content-Type") == "application/dns-udpwireformat") {
		requestBinary, err = ioutil.ReadAll(r.Body)
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
			questionClass = strconv.Itoa(int(question.Qclass))
		}
		questionType := ""
		if qtype, ok := dns.TypeToString[question.Qtype]; ok {
			questionType = qtype
		} else {
			questionType = strconv.Itoa(int(question.Qtype))
		}
		fmt.Printf("%s - - [%s] \"%s %s %s\"\n", r.RemoteAddr, time.Now().Format("02/Jan/2006:15:04:05 -0700"), questionName, questionClass, questionType)
	}

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
				ednsClientNetmask = 24
			} else {
				ednsClientFamily = 2
				ednsClientNetmask = 48
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
		request:    msg,
		isTailored: isTailored,
	}
}

func (s *Server) generateResponseIETF(w http.ResponseWriter, r *http.Request, req *DNSRequest) {
	respJSON := jsonDNS.Marshal(req.response)
	req.response.Id = 0
	respBytes, err := req.response.Pack()
	if err != nil {
		log.Println(err)
		jsonDNS.FormatError(w, fmt.Sprintf("DNS packet construct failure (%s)", err.Error()), 500)
		return
	}

	w.Header().Set("Content-Type", "application/dns-message")
	now := time.Now().UTC().Format(http.TimeFormat)
	w.Header().Set("Date", now)
	w.Header().Set("Last-Modified", now)
	if respJSON.HaveTTL {
		if req.isTailored {
			w.Header().Set("Cache-Control", "private, max-age="+strconv.Itoa(int(respJSON.LeastTTL)))
		} else {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(respJSON.LeastTTL)))
		}
		w.Header().Set("Expires", respJSON.EarliestExpires.Format(http.TimeFormat))
	}
	if respJSON.Status == dns.RcodeServerFailure {
		w.WriteHeader(503)
	}
	w.Write(respBytes)
}
