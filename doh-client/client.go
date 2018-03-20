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
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"../json-dns"
	"github.com/miekg/dns"
	"golang.org/x/net/http2"
)

type Client struct {
	conf          *config
	bootstrap     []string
	udpServer     *dns.Server
	tcpServer     *dns.Server
	httpTransport *http.Transport
	httpClient    *http.Client
}

func NewClient(conf *config) (c *Client, err error) {
	c = &Client{
		conf: conf,
	}
	c.udpServer = &dns.Server{
		Addr:    conf.Listen,
		Net:     "udp",
		Handler: dns.HandlerFunc(c.udpHandlerFunc),
		UDPSize: 4096,
	}
	c.tcpServer = &dns.Server{
		Addr:    conf.Listen,
		Net:     "tcp",
		Handler: dns.HandlerFunc(c.tcpHandlerFunc),
	}
	bootResolver := net.DefaultResolver
	if len(conf.Bootstrap) != 0 {
		c.bootstrap = make([]string, len(conf.Bootstrap))
		for i, bootstrap := range conf.Bootstrap {
			bootstrapAddr, err := net.ResolveUDPAddr("udp", bootstrap)
			if err != nil {
				bootstrapAddr, err = net.ResolveUDPAddr("udp", "["+bootstrap+"]:53")
			}
			if err != nil {
				return nil, err
			}
			c.bootstrap[i] = bootstrapAddr.String()
		}
		bootResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				numServers := len(c.bootstrap)
				bootstrap := c.bootstrap[rand.Intn(numServers)]
				conn, err := d.DialContext(ctx, network, bootstrap)
				return conn, err
			},
		}
	}
	c.httpTransport = new(http.Transport)
	c.httpTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(conf.Timeout) * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
			Resolver:  bootResolver,
		}).DialContext,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: time.Duration(conf.Timeout) * time.Second,
		TLSHandshakeTimeout:   time.Duration(conf.Timeout) * time.Second,
	}
	http2.ConfigureTransport(c.httpTransport)
	// Most CDNs require Cookie support to prevent DDoS attack
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c.httpClient = &http.Client{
		Transport: c.httpTransport,
		Jar:       cookieJar,
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
	}()
	go func() {
		err := c.tcpServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
		result <- err
	}()
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

	if len(c.conf.UpstreamIETF) == 0 {
		c.handlerFuncGoogle(w, r, isTCP)
		return
	}
	if len(c.conf.UpstreamGoogle) == 0 {
		c.handlerFuncIETF(w, r, isTCP)
		return
	}
	numServers := len(c.conf.UpstreamGoogle) + len(c.conf.UpstreamIETF)
	random := rand.Intn(numServers)
	if random < len(c.conf.UpstreamGoogle) {
		c.handlerFuncGoogle(w, r, isTCP)
	} else {
		c.handlerFuncIETF(w, r, isTCP)
	}
}

func (c *Client) udpHandlerFunc(w dns.ResponseWriter, r *dns.Msg) {
	c.handlerFunc(w, r, false)
}

func (c *Client) tcpHandlerFunc(w dns.ResponseWriter, r *dns.Msg) {
	c.handlerFunc(w, r, true)
}

var (
	ipv4Mask24 = net.IPMask{255, 255, 255, 0}
	ipv6Mask48 = net.CIDRMask(48, 128)
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
