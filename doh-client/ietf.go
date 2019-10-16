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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/m13253/dns-over-https/doh-client/selector"
	jsonDNS "github.com/m13253/dns-over-https/json-dns"
	"github.com/miekg/dns"
)

func (c *Client) generateRequestIETF(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, isTCP bool, upstream *selector.Upstream) *DNSRequest {
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
		reply := jsonDNS.PrepareReply(r)
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return &DNSRequest{
			err: err,
		}
	}
	r.Id = requestID
	requestBase64 := base64.RawURLEncoding.EncodeToString(requestBinary)

	requestURL := fmt.Sprintf("%s?ct=application/dns-message&dns=%s", upstream.URL, requestBase64)

	var req *http.Request
	if len(requestURL) < 2048 {
		req, err = http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			log.Println(err)
			reply := jsonDNS.PrepareReply(r)
			reply.Rcode = dns.RcodeServerFailure
			w.WriteMsg(reply)
			return &DNSRequest{
				err: err,
			}
		}
	} else {
		req, err = http.NewRequest(http.MethodPost, upstream.URL, bytes.NewReader(requestBinary))
		if err != nil {
			log.Println(err)
			reply := jsonDNS.PrepareReply(r)
			reply.Rcode = dns.RcodeServerFailure
			w.WriteMsg(reply)
			return &DNSRequest{
				err: err,
			}
		}
		req.Header.Set("Content-Type", "application/dns-message")
	}
	req.Header.Set("Accept", "application/dns-message, application/dns-udpwireformat, application/json")
	if !c.conf.Other.NoUserAgent {
		req.Header.Set("User-Agent", USER_AGENT)
	} else {
		req.Header.Set("User-Agent", "")
	}
	req = req.WithContext(ctx)
	c.httpClientMux.RLock()
	resp, err := c.httpClient.Do(req)
	c.httpClientMux.RUnlock()

	// if http Client.Do returns non-nil error, it always *url.Error
	/*if err == context.DeadlineExceeded {
		// Do not respond, silently fail to prevent caching of SERVFAIL
		log.Println(err)
		return &DNSRequest{
			err: err,
		}
	}*/

	if err != nil {
		log.Println(err)
		reply := jsonDNS.PrepareReply(r)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return &DNSRequest{
			err: err,
		}
	}

	return &DNSRequest{
		response:          resp,
		reply:             jsonDNS.PrepareReply(r),
		udpSize:           udpSize,
		ednsClientAddress: ednsClientAddress,
		ednsClientNetmask: ednsClientNetmask,
		currentUpstream:   upstream.URL,
	}
}

func (c *Client) parseResponseIETF(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, isTCP bool, req *DNSRequest) {
	if req.response.StatusCode != http.StatusOK {
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
		log.Printf("read error from upstream %s: %v\n", req.currentUpstream, err)
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
			log.Printf("Date header parse error from upstream %s: %v\n", req.currentUpstream, err)
		}
	}
	headerLastModified := req.response.Header.Get("Last-Modified")
	lastModified := now
	if headerLastModified != "" {
		if lastModifiedDate, err := time.Parse(http.TimeFormat, headerLastModified); err == nil {
			lastModified = lastModifiedDate
		} else {
			log.Printf("Last-Modified header parse error from upstream %s: %v\n", req.currentUpstream, err)
		}
	}
	timeDelta := now.Sub(lastModified)
	if timeDelta < 0 {
		timeDelta = 0
	}

	fullReply := new(dns.Msg)
	err = fullReply.Unpack(body)
	if err != nil {
		log.Printf("unpacking error from upstream %s: %v\n", req.currentUpstream, err)
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
		log.Printf("packing error with upstream %s: %v\n", req.currentUpstream, err)
		req.reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(req.reply)
		return
	}
	if !isTCP && len(buf) > int(req.udpSize) {
		fullReply.Truncated = true
		buf, err = fullReply.Pack()
		if err != nil {
			log.Printf("re-packing error with upstream %s: %v\n", req.currentUpstream, err)
			return
		}
		buf = buf[:req.udpSize]
	}
	_, err = w.Write(buf)
	if err != nil {
		log.Printf("failed to write to client: %v\n", err)
	}
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
