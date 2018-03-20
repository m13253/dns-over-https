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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"../json-dns"
	"github.com/miekg/dns"
)

func (c *Client) handlerFuncGoogle(w dns.ResponseWriter, r *dns.Msg, isTCP bool) {
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

	numServers := len(c.conf.UpstreamGoogle)
	upstream := c.conf.UpstreamGoogle[rand.Intn(numServers)]
	requestURL := fmt.Sprintf("%s?name=%s&type=%s", upstream, url.QueryEscape(questionName), url.QueryEscape(questionType))

	if r.CheckingDisabled {
		requestURL += "&cd=1"
	}

	udpSize := uint16(512)
	if opt := r.IsEdns0(); opt != nil {
		udpSize = opt.UDPSize()
	}

	ednsClientAddress, ednsClientNetmask := c.findClientIP(w, r)
	if ednsClientAddress != nil {
		requestURL += fmt.Sprintf("&edns_client_subnet=%s/%d", ednsClientAddress.String(), ednsClientNetmask)
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DNS-over-HTTPS/1.0 (+https://github.com/m13253/dns-over-https)")
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
		if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") {
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

	var respJSON jsonDNS.Response
	err = json.Unmarshal(body, &respJSON)
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}

	if respJSON.Status != dns.RcodeSuccess && respJSON.Comment != "" {
		log.Printf("DNS error: %s\n", respJSON.Comment)
	}

	fullReply := jsonDNS.Unmarshal(reply, &respJSON, udpSize, ednsClientNetmask)
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
