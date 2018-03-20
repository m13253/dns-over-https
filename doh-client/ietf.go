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
	"net/http"
	"strconv"
	"strings"
	"time"

	"../json-dns"
	"github.com/miekg/dns"
)

func (c *Client) handlerFuncIETF(w dns.ResponseWriter, r *dns.Msg, isTCP bool) {
	reply := jsonDNS.PrepareReply(r)

	if len(r.Question) != 1 {
		log.Println("Number of questions is not 1")
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return
	}

	question := r.Question[0]
	// knot-resolver scrambles capitalization, I think it is unfriendly to cache
	questionName := strings.ToLower(question.Name)
	questionType := ""
	if qtype, ok := dns.TypeToString[question.Qtype]; ok {
		questionType = qtype
	} else {
		questionType = strconv.Itoa(int(question.Qtype))
	}

	if c.conf.Verbose {
		fmt.Printf("%s - - [%s] \"%s IN %s\"\n", w.RemoteAddr(), time.Now().Format("02/Jan/2006:15:04:05 -0700"), questionName, questionType)
	}

	requestID := r.Id
	r.Id = 0
	question.Name = questionName
	opt := r.IsEdns0()
	udpSize := uint16(512)
	if opt == nil {
		opt = new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(4096)
		opt.SetDo(false)
		r.Extra = append(r.Extra, opt)
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
	if edns0Subnet == nil {
		ednsClientFamily := uint16(0)
		ednsClientAddress, ednsClientNetmask := c.findClientIP(w, r)
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

	requestBinary, err := r.Pack()
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return
	}
	requestBase64 := base64.RawURLEncoding.EncodeToString(requestBinary)

	numServers := len(c.conf.UpstreamIETF)
	upstream := c.conf.UpstreamIETF[rand.Intn(numServers)]
	requestURL := fmt.Sprintf("%s?ct=application/dns-udpwireformat&dns=%s", upstream, requestBase64)

	var req *http.Request
	if len(requestURL) < 2048 {
		req, err = http.NewRequest("GET", requestURL, nil)
		if err != nil {
			log.Println(err)
			reply.Rcode = dns.RcodeServerFailure
			w.WriteMsg(reply)
			return
		}
	} else {
		req, err = http.NewRequest("POST", upstream, bytes.NewReader(requestBinary))
		if err != nil {
			log.Println(err)
			reply.Rcode = dns.RcodeServerFailure
			w.WriteMsg(reply)
			return
		}
		req.Header.Set("Content-Type", "application/dns-udpwireformat")
	}
	req.Header.Set("Accept", "application/dns-udpwireformat")
	req.Header.Set("User-Agent", "DNS-over-HTTPS/1.1 (+https://github.com/m13253/dns-over-https)")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		c.httpTransport.CloseIdleConnections()
		return
	}
	if resp.StatusCode != 200 {
		log.Printf("HTTP error: %s\n", resp.Status)
		reply.Rcode = dns.RcodeServerFailure
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/dns-udpwireformat" && !strings.HasPrefix(contentType, "application/dns-udpwireformat;") {
			w.WriteMsg(reply)
			return
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}
	lastModified := resp.Header.Get("Last-Modified")
	if lastModified == "" {
		lastModified = resp.Header.Get("Date")
	}
	now := time.Now().UTC()
	lastModifiedDate, err := time.Parse(http.TimeFormat, lastModified)
	if err != nil {
		log.Println(err)
		lastModifiedDate = now
	}
	timeDelta := now.Sub(lastModifiedDate)
	if timeDelta < 0 {
		timeDelta = 0
	}

	fullReply := new(dns.Msg)
	err = fullReply.Unpack(body)
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}

	fullReply.Id = requestID
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
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}
	if !isTCP && len(buf) > int(udpSize) {
		fullReply.Truncated = true
		buf, err = fullReply.Pack()
		if err != nil {
			log.Println(err)
			return
		}
		buf = buf[:udpSize]
	}
	w.Write(buf)
}

func fixRecordTTL(rr dns.RR, delta time.Duration) dns.RR {
	rrHeader := rr.Header()
	oldTTL := time.Duration(rrHeader.Ttl) * time.Second
	newTTL := oldTTL - delta
	if newTTL > 0 {
		rrHeader.Ttl = uint32((newTTL + time.Second/2) / time.Second)
	} else {
		rrHeader.Ttl = 0
	}
	return rr
}
