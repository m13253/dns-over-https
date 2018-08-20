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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/m13253/dns-over-https/json-dns"
	"github.com/miekg/dns"
)

func (c *Client) generateRequestIETF(w dns.ResponseWriter, r *dns.Msg, isTCP bool) *DNSRequest {
	reply := jsonDNS.PrepareReply(r)

	if len(r.Question) != 1 {
		log.Println("Number of questions is not 1")
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return &DNSRequest{
			err: &dns.Error{},
		}
	}

	question := &r.Question[0]
	questionName := question.Name
	questionType := ""
	if qtype, ok := dns.TypeToString[question.Qtype]; ok {
		questionType = qtype
	} else {
		questionType = strconv.Itoa(int(question.Qtype))
	}

	if c.conf.Verbose {
		fmt.Printf("%s - - [%s] \"%s IN %s\"\n", w.RemoteAddr(), time.Now().Format("02/Jan/2006:15:04:05 -0700"), questionName, questionType)
	}

	question.Name = questionName
	opt := r.IsEdns0()
	udpSize := uint16(512)
	if opt == nil {
		opt = new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(dns.DefaultMsgSize)
		opt.SetDo(false)
		r.Extra = append([]dns.RR{opt}, r.Extra...)
	} else {
		udpSize = opt.UDPSize()
	}
	var edns0Subnet *dns.EDNS0_SUBNET
	for _, option := range opt.Option {
		if option.Option() == dns.EDNS0SUBNET {
			edns0Subnet = option.(*dns.EDNS0_SUBNET)
			break
		}
	}
	ednsClientAddress, ednsClientNetmask := net.IP(nil), uint8(255)
	if edns0Subnet == nil {
		ednsClientFamily := uint16(0)
		ednsClientAddress, ednsClientNetmask = c.findClientIP(w, r)
		if ednsClientAddress != nil {
			if ipv4 := ednsClientAddress.To4(); ipv4 != nil {
				ednsClientFamily = 1
				ednsClientAddress = ipv4
				ednsClientNetmask = 24
			} else {
				ednsClientFamily = 2
				ednsClientNetmask = 56
			}
			edns0Subnet = new(dns.EDNS0_SUBNET)
			edns0Subnet.Code = dns.EDNS0SUBNET
			edns0Subnet.Family = ednsClientFamily
			edns0Subnet.SourceNetmask = ednsClientNetmask
			edns0Subnet.SourceScope = 0
			edns0Subnet.Address = ednsClientAddress
			opt.Option = append(opt.Option, edns0Subnet)
		}
	} else {
		ednsClientAddress, ednsClientNetmask = edns0Subnet.Address, edns0Subnet.SourceNetmask
	}

	requestID := r.Id
	r.Id = 0
	requestBinary, err := r.Pack()
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return &DNSRequest{
			err: err,
		}
	}
	r.Id = requestID
	requestBase64 := base64.RawURLEncoding.EncodeToString(requestBinary)

	numServers := len(c.conf.UpstreamIETF)
	upstream := c.conf.UpstreamIETF[rand.Intn(numServers)]
	requestURL := fmt.Sprintf("%s?ct=application/dns-message&dns=%s", upstream, requestBase64)

	var req *http.Request
	if len(requestURL) < 2048 {
		req, err = http.NewRequest("GET", requestURL, nil)
		if err != nil {
			// Do not respond, silently fail to prevent caching of SERVFAIL
			log.Println(err)
			return &DNSRequest{
				err: err,
			}
		}
	} else {
		req, err = http.NewRequest("POST", upstream, bytes.NewReader(requestBinary))
		if err != nil {
			// Do not respond, silently fail to prevent caching of SERVFAIL
			log.Println(err)
			return &DNSRequest{
				err: err,
			}
		}
		req.Header.Set("Content-Type", "application/dns-message")
	}
	req.Header.Set("Accept", "application/dns-message, application/dns-udpwireformat, application/json")
	req.Header.Set("User-Agent", USER_AGENT)
	c.httpClientMux.RLock()
	resp, err := c.httpClient.Do(req)
	c.httpClientMux.RUnlock()
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		err1 := c.newHTTPClient()
		if err1 != nil {
			log.Fatalln(err1)
		}
		return &DNSRequest{
			err: err,
		}
	}

	return &DNSRequest{
		response:          resp,
		reply:             reply,
		udpSize:           udpSize,
		ednsClientAddress: ednsClientAddress,
		ednsClientNetmask: ednsClientNetmask,
		currentUpstream:   upstream,
	}
}

func (c *Client) parseResponseIETF(w dns.ResponseWriter, r *dns.Msg, isTCP bool, req *DNSRequest) {
	if req.response.StatusCode != 200 {
		log.Printf("HTTP error from upstream %s: %s\n", req.currentUpstream, req.response.Status)
		req.reply.Rcode = dns.RcodeServerFailure
		contentType := req.response.Header.Get("Content-Type")
		if contentType != "application/dns-message" && !strings.HasPrefix(contentType, "application/dns-message;") {
			w.WriteMsg(req.reply)
			return
		}
	}

	body, err := ioutil.ReadAll(req.response.Body)
	if err != nil {
		log.Println(err)
		req.reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(req.reply)
		return
	}
	headerNow := req.response.Header.Get("Date")
	now := time.Now().UTC()
	if headerNow != "" {
		if nowDate, err := time.Parse(http.TimeFormat, headerNow); err == nil {
			now = nowDate
		} else {
			log.Println(err)
		}
	}
	headerLastModified := req.response.Header.Get("Last-Modified")
	lastModified := now
	if headerLastModified != "" {
		if lastModifiedDate, err := time.Parse(http.TimeFormat, headerLastModified); err == nil {
			lastModified = lastModifiedDate
		} else {
			log.Println(err)
		}
	}
	timeDelta := now.Sub(lastModified)
	if timeDelta < 0 {
		timeDelta = 0
	}

	fullReply := new(dns.Msg)
	err = fullReply.Unpack(body)
	if err != nil {
		log.Println(err)
		req.reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(req.reply)
		return
	}

	fullReply.Id = r.Id
	for _, rr := range fullReply.Answer {
		_ = fixRecordTTL(rr, timeDelta)
	}
	for _, rr := range fullReply.Ns {
		_ = fixRecordTTL(rr, timeDelta)
	}
	for _, rr := range fullReply.Extra {
		if rr.Header().Rrtype == dns.TypeOPT {
			continue
		}
		_ = fixRecordTTL(rr, timeDelta)
	}

	buf, err := fullReply.Pack()
	if err != nil {
		log.Println(err)
		req.reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(req.reply)
		return
	}
	if !isTCP && len(buf) > int(req.udpSize) {
		fullReply.Truncated = true
		buf, err = fullReply.Pack()
		if err != nil {
			log.Println(err)
			return
		}
		buf = buf[:req.udpSize]
	}
	w.Write(buf)
}

func fixRecordTTL(rr dns.RR, delta time.Duration) dns.RR {
	rrHeader := rr.Header()
	oldTTL := time.Duration(rrHeader.Ttl) * time.Second
	newTTL := oldTTL - delta
	if newTTL > 0 {
		rrHeader.Ttl = uint32(newTTL / time.Second)
	} else {
		rrHeader.Ttl = 0
	}
	return rr
}
