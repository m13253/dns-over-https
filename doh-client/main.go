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
	"flag"
	"log"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:53", "DNS listen port")
	upstream := flag.String("upstream", "https://dns.google.com/resolve", "HTTP path for upstream resolver")
	bootstrap := flag.String("bootstrap", "", "The bootstrap DNS server to resolve the address of the upstream resolver")
	timeout := flag.Uint("timeout", 10, "Timeout for upstream request")
	noECS := flag.Bool("no-ecs", false, "Disable EDNS0-Client-Subnet, do not send client's IP address")
	verbose := flag.Bool("verbose", false, "Enable logging")
	flag.Parse()

	bootstraps := []string {}
	if *bootstrap != "" {
		bootstraps = strings.Split(*bootstrap, ",")
	}
	client, err := NewClient(*addr, *upstream, bootstraps, *timeout, *noECS, *verbose)
	if err != nil { log.Fatalln(err) }
	_ = client.Start()
}
