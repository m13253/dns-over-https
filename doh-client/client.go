/*
    DNS-over-HTTPS
    Copyright (C) 2017 Star Brilliant <m13253@hotmail.com>

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
	"math/rand"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
	"github.com/miekg/dns"
	"../json-dns"
)

type Client struct {
	conf			*config
	bootstrap		[]string
	udpServer		*dns.Server
	tcpServer		*dns.Server
	httpTransport	*http.Transport
	httpClient		*http.Client
}

func NewClient(conf *config) (c *Client, err error) {
	c = &Client {
		conf: conf,
	}
	c.udpServer = &dns.Server {
		Addr: conf.Listen,
		Net: "udp",
		Handler: dns.HandlerFunc(c.udpHandlerFunc),
		UDPSize: 4096,
	}
	c.tcpServer = &dns.Server {
		Addr: conf.Listen,
		Net: "tcp",
		Handler: dns.HandlerFunc(c.tcpHandlerFunc),
	}
	bootResolver := net.DefaultResolver
	if len(conf.Bootstrap) != 0 {
		c.bootstrap = make([]string, len(conf.Bootstrap))
		for i, bootstrap := range conf.Bootstrap {
			bootstrapAddr, err := net.ResolveUDPAddr("udp", bootstrap)
			if err != nil {
				bootstrapAddr, err = net.ResolveUDPAddr("udp", "[" + bootstrap + "]:53")
			}
			if err != nil { return nil, err }
			c.bootstrap[i] = bootstrapAddr.String()
		}
		bootResolver = &net.Resolver {
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				num_servers := len(c.bootstrap)
				bootstrap := c.bootstrap[rand.Intn(num_servers)]
				conn, err := d.DialContext(ctx, network, bootstrap)
				return conn, err
			},
		}
	}
	c.httpTransport = new(http.Transport)
	*c.httpTransport = *http.DefaultTransport.(*http.Transport)
	c.httpTransport.DialContext = (&net.Dialer {
		Timeout: time.Duration(conf.Timeout) * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
		Resolver: bootResolver,
	}).DialContext
	c.httpTransport.ResponseHeaderTimeout = time.Duration(conf.Timeout) * time.Second
	// Most CDNs require Cookie support to prevent DDoS attack
	cookieJar, err := cookiejar.New(nil)
	if err != nil { return nil, err }
	c.httpClient = &http.Client {
		Transport: c.httpTransport,
		Jar: cookieJar,
	}
	return c, nil
}

func (c *Client) Start() error {
	result := make(chan error)
	go func() {
		err := c.udpServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
		result <- err
	} ()
	go func() {
		err := c.tcpServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
		result <- err
	} ()
	err := <-result
	if err != nil {
		return err
	}
	err = <-result
	return err
}

func (c *Client) handlerFunc(w dns.ResponseWriter, r *dns.Msg, isTCP bool) {
	if r.Response == true {
		log.Println("Received a response packet")
		return
	}

	reply := jsonDNS.PrepareReply(r)

	if len(r.Question) != 1 {
		log.Println("Number of questions is not 1")
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return
	}
	question := r.Question[0]
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

	num_servers := len(c.conf.Upstream)
	upstream := c.conf.Upstream[rand.Intn(num_servers)]
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
		w.WriteMsg(reply)
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") {
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

	var respJson jsonDNS.Response
	err = json.Unmarshal(body, &respJson)
	if err != nil {
		log.Println(err)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}

	if respJson.Status != dns.RcodeSuccess && respJson.Comment != "" {
		log.Printf("DNS error: %s\n", respJson.Comment)
	}

	fullReply := jsonDNS.Unmarshal(reply, &respJson, udpSize, ednsClientNetmask)
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

func (c *Client) udpHandlerFunc(w dns.ResponseWriter, r *dns.Msg) {
	c.handlerFunc(w, r, false)
}

func (c *Client) tcpHandlerFunc(w dns.ResponseWriter, r *dns.Msg) {
	c.handlerFunc(w, r, true)
}

var (
	ipv4Mask24	net.IPMask = net.IPMask { 255, 255, 255, 0 }
	ipv6Mask48	net.IPMask = net.CIDRMask(48, 128)
)

func (c *Client) findClientIP(w dns.ResponseWriter, r *dns.Msg) (ednsClientAddress net.IP, ednsClientNetmask uint8) {
	ednsClientNetmask = 255
	if c.conf.NoECS {
		return net.IPv4(0, 0, 0, 0), 0
	}
	if opt := r.IsEdns0(); opt != nil {
		for _, option := range opt.Option {
			if option.Option() == dns.EDNS0SUBNET {
				edns0Subnet := option.(*dns.EDNS0_SUBNET)
				ednsClientAddress = edns0Subnet.Address
				ednsClientNetmask = edns0Subnet.SourceNetmask
				return
			}
		}
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", w.RemoteAddr().String())
	if err != nil {
		return
	}
	if ip := remoteAddr.IP; jsonDNS.IsGlobalIP(ip) {
		if ipv4 := ip.To4(); ipv4 != nil {
			ednsClientAddress = ipv4.Mask(ipv4Mask24)
			ednsClientNetmask = 24
		} else {
			ednsClientAddress = ip.Mask(ipv6Mask48)
			ednsClientNetmask = 48
		}
	}
	return
}
