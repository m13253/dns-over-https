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

package jsonDNS

import (
	"net"
)

// RFC6890
var localIPv4Nets = []net.IPNet{
	// This host on this network
	net.IPNet{
		net.IP{0, 0, 0, 0},
		net.IPMask{255, 0, 0, 0},
	},
	// Private-Use Networks
	net.IPNet{
		net.IP{10, 0, 0, 0},
		net.IPMask{255, 0, 0, 0},
	},
	// Shared Address Space
	net.IPNet{
		net.IP{100, 64, 0, 0},
		net.IPMask{255, 192, 0, 0},
	},
	// Loopback
	net.IPNet{
		net.IP{127, 0, 0, 0},
		net.IPMask{255, 0, 0, 0},
	},
	// Link Local
	net.IPNet{
		net.IP{169, 254, 0, 0},
		net.IPMask{255, 255, 0, 0},
	},
	// Private-Use Networks
	net.IPNet{
		net.IP{172, 16, 0, 0},
		net.IPMask{255, 240, 0, 0},
	},
	// DS-Lite
	net.IPNet{
		net.IP{192, 0, 0, 0},
		net.IPMask{255, 255, 255, 248},
	},
	// 6to4 Relay Anycast
	net.IPNet{
		net.IP{192, 88, 99, 0},
		net.IPMask{255, 255, 255, 0},
	},
	// Private-Use Networks
	net.IPNet{
		net.IP{192, 168, 0, 0},
		net.IPMask{255, 255, 0, 0},
	},
	// Reserved for Future Use & Limited Broadcast
	net.IPNet{
		net.IP{240, 0, 0, 0},
		net.IPMask{240, 0, 0, 0},
	},
}

// RFC6890
var localIPv6Nets = []net.IPNet{
	// Unspecified & Loopback Address
	net.IPNet{
		net.IP{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe},
	},
	// Discard-Only Prefix
	net.IPNet{
		net.IP{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
	// Unique-Local
	net.IPNet{
		net.IP{0xfc, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xfe, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
	// Linked-Scoped Unicast
	net.IPNet{
		net.IP{0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xff, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
}

func IsGlobalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ipv4 := ip.To4(); len(ipv4) == net.IPv4len {
		for _, ipnet := range localIPv4Nets {
			if ipnet.Contains(ip) {
				return false
			}
		}
		return true
	}
	if len(ip) == net.IPv6len {
		for _, ipnet := range localIPv6Nets {
			if ipnet.Contains(ip) {
				return false
			}
		}
		return true
	}
	return true
}
