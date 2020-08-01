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
	"github.com/infobloxopen/go-trees/iptree"
	"net"
)

var defaultFilter *iptree.Tree

func init() {
	defaultFilter = iptree.NewTree()

	// RFC6890
	// This host on this network
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{0, 0, 0, 0},
		net.IPMask{255, 0, 0, 0},
	}, struct{}{})

	// Private-Use Networks
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{10, 0, 0, 0},
		net.IPMask{255, 0, 0, 0},
	}, struct{}{})

	// Shared Address Space
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{100, 64, 0, 0},
		net.IPMask{255, 192, 0, 0},
	}, struct{}{})

	// Loopback
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{127, 0, 0, 0},
		net.IPMask{255, 0, 0, 0},
	}, struct{}{})

	// Link Local
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{169, 254, 0, 0},
		net.IPMask{255, 255, 0, 0},
	}, struct{}{})

	// Private-Use Networks
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{172, 16, 0, 0},
		net.IPMask{255, 240, 0, 0},
	}, struct{}{})

	// DS-Lite
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{192, 0, 0, 0},
		net.IPMask{255, 255, 255, 248},
	}, struct{}{})

	// 6to4 Relay Anycast
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{192, 88, 99, 0},
		net.IPMask{255, 255, 255, 0},
	}, struct{}{})

	// Private-Use Networks
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{192, 168, 0, 0},
		net.IPMask{255, 255, 0, 0},
	}, struct{}{})

	// Reserved for Future Use & Limited Broadcast
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{240, 0, 0, 0},
		net.IPMask{240, 0, 0, 0},
	}, struct{}{})

	// RFC6890
	// Unspecified & Loopback Address
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe},
	}, struct{}{})

	// Discard-Only Prefix
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}, struct{}{})

	// Unique-Local
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{0xfc, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xfe, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}, struct{}{})

	// Linked-Scoped Unicast
	defaultFilter.InplaceInsertNet(&net.IPNet{
		net.IP{0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		net.IPMask{0xff, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}, struct{}{})

}

func IsGlobalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	_, contained := defaultFilter.GetByIP(ip)
	return !contained
}
