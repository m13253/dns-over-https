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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/m13253/dns-over-https/doh-client/selector"
	"github.com/m13253/dns-over-https/json-dns"
	"github.com/miekg/dns"
)

func (c *Client) generateRequestGoogle(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, isTCP bool, upstream *selector.Upstream) *DNSRequest {
	question := &r.Question[0]
	questionName := question.Name
	questionClass := question.Qclass
	if questionClass != dns.ClassINET {
		reply := jsonDNS.PrepareReply(r)
		reply.Rcode = dns.RcodeRefused
		w.WriteMsg(reply)
		return &DNSRequest{
			err: &dns.Error{},
		}
	}
	questionType := ""
	if qtype, ok := dns.TypeToString[question.Qtype]; ok {
		questionType = qtype
	} else {
		questionType = strconv.FormatUint(uint64(question.Qtype), 10)
	}

	requestURL := fmt.Sprintf("%s?ct=application/dns-json&name=%s&type=%s", upstream.URL, url.QueryEscape(questionName), url.QueryEscape(questionType))

	if r.CheckingDisabled {
		requestURL += "&cd=1"
	}

	udpSize := uint16(512)
	if opt := r.IsEdns0(); opt != nil {
		udpSize = opt.UDPSize()
		if opt.Do() {
			requestURL += "&do=1"
		}
	}

	ednsClientAddress, ednsClientNetmask := c.findClientIP(w, r)
	if ednsClientAddress != nil {
		requestURL += fmt.Sprintf("&edns_client_subnet=%s/%d", ednsClientAddress.String(), ednsClientNetmask)
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		log.Println(err)
		reply := jsonDNS.PrepareReply(r)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return &DNSRequest{
			err: err,
		}
	}

	req.Header.Set("Accept", "application/json, application/dns-message, application/dns-udpwireformat")
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

func (c *Client) parseResponseGoogle(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, isTCP bool, req *DNSRequest) {
	if req.response.StatusCode != http.StatusOK {
		log.Printf("HTTP error from upstream %s: %s\n", req.currentUpstream, req.response.Status)
		req.reply.Rcode = dns.RcodeServerFailure
		contentType := req.response.Header.Get("Content-Type")
		if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") {
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

	var respJSON jsonDNS.Response
	err = json.Unmarshal(body, &respJSON)
	if err != nil {
		log.Println(err)
		req.reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(req.reply)
		return
	}

	if respJSON.Status != dns.RcodeSuccess && respJSON.Comment != "" {
		log.Printf("DNS error: %s\n", respJSON.Comment)
	}
	fixEmptyNames(&respJSON)

	fullReply := jsonDNS.Unmarshal(req.reply, &respJSON, req.udpSize, req.ednsClientNetmask)
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

// Fix DNS response empty []RR.Name
// Additional section won't be rectified
// see: https://stackoverflow.com/questions/52136176/what-is-additional-section-in-dns-and-how-it-works
func fixEmptyNames(respJSON *jsonDNS.Response) {
	for i := range respJSON.Answer {
		if respJSON.Answer[i].Name == "" {
			respJSON.Answer[i].Name = "."
		}
	}
	for i := range respJSON.Authority {
		if respJSON.Authority[i].Name == "" {
			respJSON.Authority[i].Name = "."
		}
	}
}
