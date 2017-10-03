/*
    DNS-over-HTTPS
    Copyright (C) 2017 Star Brilliant <m13253@hotmail.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published
    by the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	addr := flag.String("addr", "[::1]:8080", "HTTP listen port")
	cert := flag.String("cert", "", "TLS certification file")
	key := flag.String("key", "", "TLS key file")
	path := flag.String("path", "/resolve", "HTTP path for resolve application")
	upstream := flag.String("upstream", "8.8.8.8:53,8.8.4.4:53", "Upstream DNS resolver")
	tcpOnly := flag.Bool("tcp", false, "Only use TCP for DNS query")
	flag.Parse()

	if (*cert != "") != (*key != "") {
		log.Fatalln("You must specify both -cert and -key to enable TLS")
	}

	upstreams := strings.Split(*upstream, ",")
	server := NewServer(*addr, *cert, *key, *path, upstreams, *tcpOnly)
	err := server.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
