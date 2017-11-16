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
)

func main() {
	addr := flag.String("addr", "127.0.0.1:53", "DNS listen port")
	upstream := flag.String("upstream", "https://dns.google.com/resolve", "HTTP path for upstream resolver")
	bootstrap := flag.String("bootstrap", "", "The bootstrap DNS server to resolve the address of the upstream resolver")
	timeout := flag.Uint("timeout", 10, "Timeout for upstream request")
	flag.Parse()

	client, err := NewClient(*addr, *upstream, *bootstrap, *timeout)
	if err != nil { log.Fatalln(err) }
	_ = client.Start()
}
