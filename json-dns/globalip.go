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

package jsonDNS

import (
	"net"
)

// RFC6890
var localIPv4Nets = []net.IPNet {
	// This host on this network
	net.IPNet {
		net.IP { 0, 0, 0, 0 },
		net.IPMask { 255, 0, 0, 0 },
	},
	// Private-Use Networks
	net.IPNet {
		net.IP { 10, 0, 0, 0 },
		net.IPMask { 255, 0, 0, 0 },
	},
	// Shared Address Space
	net.IPNet {
		net.IP { 100, 64, 0, 0 },
		net.IPMask { 255, 192, 0, 0 },
	},
	// Loopback
	net.IPNet {
		net.IP { 127, 0, 0, 0 },
		net.IPMask { 255, 0, 0, 0 },
	},
	// Link Local
	net.IPNet {
		net.IP { 169, 254, 0, 0 },
		net.IPMask { 255, 255, 0, 0 },
	},
	// Private-Use Networks
	net.IPNet {
		net.IP { 172, 16, 0, 0 },
		net.IPMask { 255, 240, 0, 0 },
	},
	// DS-Lite
	net.IPNet {
		net.IP { 192, 0, 0, 0 },
		net.IPMask { 255, 255, 255, 248 },
	},
	// 6to4 Relay Anycast
	net.IPNet {
		net.IP { 192, 88, 99, 0 },
		net.IPMask { 255, 255, 255, 0 },
	},
	// Private-Use Networks
	net.IPNet {
		net.IP { 192, 168, 0, 0 },
		net.IPMask { 255, 255, 0, 0 },
	},
	// Reserved for Future Use & Limited Broadcast
	net.IPNet {
		net.IP { 240, 0, 0, 0 },
		net.IPMask { 240, 0, 0, 0 },
	},
}

// RFC6890
var localIPv6Nets = []net.IPNet {
	// Unspecified & Loopback Address
	net.IPNet {
		net.IP { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
		net.IPMask { 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe },
	},
	// Discard-Only Prefix
	net.IPNet {
		net.IP { 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
		net.IPMask { 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
	},
	// Unique-Local
	net.IPNet {
		net.IP { 0xfc, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
		net.IPMask { 0xfe, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
	},
	// Linked-Scoped Unicast
	net.IPNet {
		net.IP { 0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
		net.IPMask { 0xff, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 },
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
