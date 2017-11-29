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
	addr := flag.String("addr", "127.0.0.1:8053", "HTTP listen port")
	cert := flag.String("cert", "", "TLS certification file")
	key := flag.String("key", "", "TLS key file")
	path := flag.String("path", "/resolve", "HTTP path for resolve application")
	upstream := flag.String("upstream", "8.8.8.8:53,8.8.4.4:53", "Upstream DNS resolver")
	tcpOnly := flag.Bool("tcp", false, "Only use TCP for DNS query")
	verbose := flag.Bool("verbose", false, "Enable logging")
	flag.Parse()

	if (*cert != "") != (*key != "") {
		log.Fatalln("You must specify both -cert and -key to enable TLS")
	}

	upstreams := strings.Split(*upstream, ",")
	server := NewServer(*addr, *cert, *key, *path, upstreams, *tcpOnly, *verbose)
	err := server.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
