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
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/http2"
	"golang.org/x/net/idna"

	"github.com/m13253/dns-over-https/v2/doh-client/config"
	"github.com/m13253/dns-over-https/v2/doh-client/selector"
	jsondns "github.com/m13253/dns-over-https/v2/json-dns"
)

type Client struct {
	httpClientLastCreate time.Time
	cookieJar            http.CookieJar
	selector             selector.Selector
	httpClientMux        *sync.RWMutex
	tcpClient            *dns.Client
	bootstrapResolver    *net.Resolver
	udpClient            *dns.Client
	conf                 *config.Config
	httpTransport        *http.Transport
	httpClient           *http.Client
	udpServers           []*dns.Server
	tcpServers           []*dns.Server
	passthrough          []string
	bootstrap            []string
}

type DNSRequest struct {
	err               error
	response          *http.Response
	reply             *dns.Msg
	currentUpstream   string
	ednsClientAddress net.IP
	udpSize           uint16
	ednsClientNetmask uint8
}

func NewClient(conf *config.Config) (c *Client, err error) {
	c = &Client{
		conf: conf,
	}

	udpHandler := dns.HandlerFunc(c.udpHandlerFunc)
	tcpHandler := dns.HandlerFunc(c.tcpHandlerFunc)
	c.udpClient = &dns.Client{
		Net:     "udp",
		UDPSize: dns.DefaultMsgSize,
		Timeout: time.Duration(conf.Other.Timeout) * time.Second,
	}
	c.tcpClient = &dns.Client{
		Net:     "tcp",
		Timeout: time.Duration(conf.Other.Timeout) * time.Second,
	}

	if c.conf.Other.Interface != "" {
		localV4, localV6, err := c.getInterfaceIPs()
		if err != nil {
			return nil, fmt.Errorf("failed to get interface IPs for %s: %v", c.conf.Other.Interface, err)
		}
		var localAddr net.IP
		if localV4 != nil {
			localAddr = localV4
		} else {
			localAddr = localV6
		}

		c.udpClient.Dialer = &net.Dialer{
			Timeout:   time.Duration(conf.Other.Timeout) * time.Second,
			LocalAddr: &net.UDPAddr{IP: localAddr},
		}
		c.tcpClient.Dialer = &net.Dialer{
			Timeout:   time.Duration(conf.Other.Timeout) * time.Second,
			LocalAddr: &net.TCPAddr{IP: localAddr},
		}
	}

	for _, addr := range conf.Listen {
		c.udpServers = append(c.udpServers, &dns.Server{
			Addr:    addr,
			Net:     "udp",
			Handler: udpHandler,
			UDPSize: dns.DefaultMsgSize,
		})
		c.tcpServers = append(c.tcpServers, &dns.Server{
			Addr:    addr,
			Net:     "tcp",
			Handler: tcpHandler,
		})
	}
	c.bootstrapResolver = net.DefaultResolver
	if len(conf.Other.Bootstrap) != 0 {
		c.bootstrap = make([]string, len(conf.Other.Bootstrap))
		for i, bootstrap := range conf.Other.Bootstrap {
			bootstrapAddr, err := net.ResolveUDPAddr("udp", bootstrap)
			if err != nil {
				bootstrapAddr, err = net.ResolveUDPAddr("udp", "["+bootstrap+"]:53")
			}
			if err != nil {
				return nil, err
			}
			c.bootstrap[i] = bootstrapAddr.String()
		}
		c.bootstrapResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				if c.conf.Other.Interface != "" {
					localV4, localV6, err := c.getInterfaceIPs()
					if err != nil {
						log.Printf("Bootstrap dial warning: %v", err)
					} else {
						numServers := len(c.bootstrap)
						bootstrap := c.bootstrap[rand.Intn(numServers)]
						host, _, _ := net.SplitHostPort(bootstrap)
						ip := net.ParseIP(host)
						if ip != nil {
							if ip.To4() != nil {
								if localV4 != nil {
									if strings.HasPrefix(network, "udp") {
										d.LocalAddr = &net.UDPAddr{IP: localV4}
									} else {
										d.LocalAddr = &net.TCPAddr{IP: localV4}
									}
								}
							} else {
								if localV6 != nil {
									if strings.HasPrefix(network, "udp") {
										d.LocalAddr = &net.UDPAddr{IP: localV6}
									} else {
										d.LocalAddr = &net.TCPAddr{IP: localV6}
									}
								}
							}
						}
						conn, err := d.DialContext(ctx, network, bootstrap)
						return conn, err
					}
				}
				numServers := len(c.bootstrap)
				bootstrap := c.bootstrap[rand.Intn(numServers)]
				conn, err := d.DialContext(ctx, network, bootstrap)
				return conn, err
			},
		}
		if len(conf.Other.Passthrough) != 0 {
			c.passthrough = make([]string, len(conf.Other.Passthrough))
			for i, passthrough := range conf.Other.Passthrough {
				if punycode, err := idna.ToASCII(passthrough); err != nil {
					passthrough = punycode
				}
				c.passthrough[i] = "." + strings.ToLower(strings.Trim(passthrough, ".")) + "."
			}
		}
	}
	// Most CDNs require Cookie support to prevent DDoS attack.
	// Disabling Cookie does not effectively prevent tracking,
	// so I will leave it on to make anti-DDoS services happy.
	if !c.conf.Other.NoCookies {
		c.cookieJar, err = cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
	} else {
		c.cookieJar = nil
	}

	c.httpClientMux = new(sync.RWMutex)
	err = c.newHTTPClient()
	if err != nil {
		return nil, err
	}

	switch c.conf.Upstream.UpstreamSelector {
	case config.NginxWRR:
		if c.conf.Other.Verbose {
			log.Println(config.NginxWRR, "mode start")
		}

		s := selector.NewNginxWRRSelector(time.Duration(c.conf.Other.Timeout) * time.Second)
		for _, u := range c.conf.Upstream.UpstreamGoogle {
			if err := s.Add(u.URL, selector.Google, u.Weight); err != nil {
				return nil, err
			}
		}

		for _, u := range c.conf.Upstream.UpstreamIETF {
			if err := s.Add(u.URL, selector.IETF, u.Weight); err != nil {
				return nil, err
			}
		}

		c.selector = s

	case config.LVSWRR:
		if c.conf.Other.Verbose {
			log.Println(config.LVSWRR, "mode start")
		}

		s := selector.NewLVSWRRSelector(time.Duration(c.conf.Other.Timeout) * time.Second)
		for _, u := range c.conf.Upstream.UpstreamGoogle {
			if err := s.Add(u.URL, selector.Google, u.Weight); err != nil {
				return nil, err
			}
		}

		for _, u := range c.conf.Upstream.UpstreamIETF {
			if err := s.Add(u.URL, selector.IETF, u.Weight); err != nil {
				return nil, err
			}
		}

		c.selector = s
	case config.Hash:
	    if c.conf.Other.Verbose {
		log.Println(config.Random, "mode start")
	    }
	    s := selector.NewHashSelector()
	    for _, u := range c.conf.Upstream.UpstreamGoogle {
		if err := s.Add(u.URL, selector.Google); err != nil {
		    return nil, err
		}
	    }
	    
	    for _, u := range c.conf.Upstream.UpstreamIETF {
		if err := s.Add(u.URL, selector.IETF); err != nil {
		    return nil, err
		}
	    }
	    
	    c.selector = s
	    
	default:
		if c.conf.Other.Verbose {
			log.Println(config.Random, "mode start")
		}

		// if selector is invalid or random, use random selector, or should we stop program and let user knows he is wrong?
		s := selector.NewRandomSelector()
		for _, u := range c.conf.Upstream.UpstreamGoogle {
			if err := s.Add(u.URL, selector.Google); err != nil {
				return nil, err
			}
		}

		for _, u := range c.conf.Upstream.UpstreamIETF {
			if err := s.Add(u.URL, selector.IETF); err != nil {
				return nil, err
			}
		}

		c.selector = s
	}

	if c.conf.Other.Verbose {
		if reporter, ok := c.selector.(selector.DebugReporter); ok {
			reporter.ReportWeights()
		}
	}

	return c, nil
}

func (c *Client) newHTTPClient() error {
	c.httpClientMux.Lock()
	defer c.httpClientMux.Unlock()
	if !c.httpClientLastCreate.IsZero() && time.Since(c.httpClientLastCreate) < time.Duration(c.conf.Other.Timeout)*time.Second {
		return nil
	}
	if c.httpTransport != nil {
		c.httpTransport.CloseIdleConnections()
	}

	localV4, localV6, err := c.getInterfaceIPs()
	if err != nil {
		log.Printf("Interface binding error: %v", err)
		return err
	}

	baseDialer := &net.Dialer{
		Timeout:   time.Duration(c.conf.Other.Timeout) * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  c.bootstrapResolver,
	}

	c.httpTransport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if c.conf.Other.Interface == "" {
				return baseDialer.DialContext(ctx, network, addr)
			}

			if network == "tcp4" && localV4 != nil {
				d := *baseDialer
				d.LocalAddr = &net.TCPAddr{IP: localV4}
				return d.DialContext(ctx, network, addr)
			}
			if network == "tcp6" && localV6 != nil {
				d := *baseDialer
				d.LocalAddr = &net.TCPAddr{IP: localV6}
				return d.DialContext(ctx, network, addr)
			}

			// Manual Dual-Stack: Resolve host and try compatible families sequentially
			host, port, _ := net.SplitHostPort(addr)
			ips, err := c.bootstrapResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}

			var lastErr error
			for _, ip := range ips {
				d := *baseDialer
				targetAddr := net.JoinHostPort(ip.String(), port)

				if ip.IP.To4() != nil {
					if localV4 == nil {
						continue
					}
					d.LocalAddr = &net.TCPAddr{IP: localV4}
				} else {
					if localV6 == nil {
						continue
					}
					d.LocalAddr = &net.TCPAddr{IP: localV6}
				}

				conn, err := d.DialContext(ctx, "tcp", targetAddr)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}

			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("connection to %s failed: no matching local/remote IP families on interface %s", addr, c.conf.Other.Interface)
		},
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   time.Duration(c.conf.Other.Timeout) * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: c.conf.Other.TLSInsecureSkipVerify},
	}

	if c.conf.Other.NoIPv6 {
		originalDial := c.httpTransport.DialContext
		c.httpTransport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			if strings.HasPrefix(network, "tcp") {
				network = "tcp4"
			}
			return originalDial(ctx, network, address)
		}
	}

	err = http2.ConfigureTransport(c.httpTransport)
	if err != nil {
		return err
	}
	c.httpClient = &http.Client{
		Transport: c.httpTransport,
		Jar:       c.cookieJar,
	}
	c.httpClientLastCreate = time.Now()
	return nil
}

func (c *Client) Start() error {
	results := make(chan error, len(c.udpServers)+len(c.tcpServers))
	for _, srv := range append(c.udpServers, c.tcpServers...) {
		go func(srv *dns.Server) {
			err := srv.ListenAndServe()
			if err != nil {
				log.Println(err)
			}
			results <- err
		}(srv)
	}

	// start evaluation loop
	c.selector.StartEvaluate()

	for i := 0; i < cap(results); i++ {
		err := <-results
		if err != nil {
			return err
		}
	}
	close(results)

	return nil
}

func (c *Client) handlerFunc(w dns.ResponseWriter, r *dns.Msg, isTCP bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.conf.Other.Timeout)*time.Second)
	defer cancel()

	if r.Response {
		log.Println("Received a response packet")
		return
	}

	if len(r.Question) != 1 {
		log.Println("Number of questions is not 1")
		reply := jsondns.PrepareReply(r)
		reply.Rcode = dns.RcodeFormatError
		w.WriteMsg(reply)
		return
	}
	question := &r.Question[0]
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
	if c.conf.Other.Verbose {
		fmt.Printf("%s - - [%s] \"%s %s %s\"\n", w.RemoteAddr(), time.Now().Format("02/Jan/2006:15:04:05 -0700"), questionName, questionClass, questionType)
	}

	shouldPassthrough := false
	passthroughQuestionName := questionName
	if punycode, err := idna.ToASCII(passthroughQuestionName); err != nil {
		passthroughQuestionName = punycode
	}
	passthroughQuestionName = "." + strings.ToLower(strings.Trim(passthroughQuestionName, ".")) + "."
	for _, passthrough := range c.passthrough {
		if strings.HasSuffix(passthroughQuestionName, passthrough) {
			shouldPassthrough = true
			break
		}
	}
	if shouldPassthrough {
		numServers := len(c.bootstrap)
		upstream := c.bootstrap[rand.Intn(numServers)]
		log.Printf("Request \"%s %s %s\" is passed through %s.\n", questionName, questionClass, questionType, upstream)
		var reply *dns.Msg
		var err error
		if !isTCP {
			reply, _, err = c.udpClient.Exchange(r, upstream)
		} else {
			reply, _, err = c.tcpClient.Exchange(r, upstream)
		}
		if err == nil {
			w.WriteMsg(reply)
			return
		}
		log.Println(err)
		reply = jsondns.PrepareReply(r)
		reply.Rcode = dns.RcodeServerFailure
		w.WriteMsg(reply)
		return
	}

	upstream := c.selector.Get()
	requestType := upstream.RequestType

	if c.conf.Other.Verbose {
		log.Println("choose upstream:", upstream)
	}

	var req *DNSRequest
	switch requestType {
	case "application/dns-json":
		req = c.generateRequestGoogle(ctx, w, r, isTCP, upstream)

	case "application/dns-message":
		req = c.generateRequestIETF(ctx, w, r, isTCP, upstream)

	default:
		panic("Unknown request Content-Type")
	}

	if req.err != nil {
		if urlErr, ok := req.err.(*url.Error); ok {
			// should we only check timeout?
			if urlErr.Timeout() {
				c.selector.ReportUpstreamStatus(upstream, selector.Timeout)
			}
		}

		return
	}

	// if req.err == nil, req.response != nil
	defer req.response.Body.Close()

	for _, header := range c.conf.Other.DebugHTTPHeaders {
		if value := req.response.Header.Get(header); value != "" {
			log.Printf("%s: %s\n", header, value)
		}
	}

	candidateType := strings.SplitN(req.response.Header.Get("Content-Type"), ";", 2)[0]

	switch candidateType {
	case "application/json":
		c.parseResponseGoogle(ctx, w, r, isTCP, req)

	case "application/dns-message", "application/dns-udpwireformat":
		c.parseResponseIETF(ctx, w, r, isTCP, req)

	default:
		switch requestType {
		case "application/dns-json":
			c.parseResponseGoogle(ctx, w, r, isTCP, req)

		case "application/dns-message":
			c.parseResponseIETF(ctx, w, r, isTCP, req)

		default:
			panic("Unknown response Content-Type")
		}
	}

	// https://developers.cloudflare.com/1.1.1.1/dns-over-https/request-structure/ says
	// returns code will be 200 / 400 / 413 / 415 / 504, some server will return 503, so
	// I think if status code is 5xx, upstream must have some problems
	/*if req.response.StatusCode/100 == 5 {
		c.selector.ReportUpstreamStatus(upstream, selector.Medium)
	}*/

	switch req.response.StatusCode / 100 {
	case 5:
		c.selector.ReportUpstreamStatus(upstream, selector.Error)

	case 2:
		c.selector.ReportUpstreamStatus(upstream, selector.OK)
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
	ipv6Mask56 = net.CIDRMask(56, 128)
)

func (c *Client) findClientIP(w dns.ResponseWriter, r *dns.Msg) (ednsClientAddress net.IP, ednsClientNetmask uint8) {
	ednsClientNetmask = 255
	if c.conf.Other.NoECS {
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
	if ip := remoteAddr.IP; jsondns.IsGlobalIP(ip) {
		if ipv4 := ip.To4(); ipv4 != nil {
			ednsClientAddress = ipv4.Mask(ipv4Mask24)
			ednsClientNetmask = 24
		} else {
			ednsClientAddress = ip.Mask(ipv6Mask56)
			ednsClientNetmask = 56
		}
	}
	return
}

// getInterfaceIPs returns the first valid IPv4 and IPv6 addresses found on the interface
func (c *Client) getInterfaceIPs() (v4, v6 net.IP, err error) {
	if c.conf.Other.Interface == "" {
		return nil, nil, nil
	}
	ifi, err := net.InterfaceByName(c.conf.Other.Interface)
	if err != nil {
		return nil, nil, err
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, nil, err
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			if v4 == nil {
				v4 = ip4
			}
		} else {
			if v6 == nil && !c.conf.Other.NoIPv6 {
				v6 = ip
			}
		}
	}
	if v4 == nil && v6 == nil {
		return nil, nil, fmt.Errorf("no valid IP addresses found on interface %s", c.conf.Other.Interface)
	}
	return v4, v6, nil
}
