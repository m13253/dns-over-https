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
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	"github.com/miekg/dns"
)

func PrepareReply(req *dns.Msg) *dns.Msg {
    reply := new(dns.Msg)
    reply.Id = req.Id
    reply.Response = true
    reply.Opcode = reply.Opcode
	reply.RecursionDesired = req.RecursionDesired
	reply.CheckingDisabled = req.CheckingDisabled
	reply.Rcode = dns.RcodeServerFailure
    reply.Compress = true
	reply.Question = make([]dns.Question, len(req.Question))
	copy(reply.Question, req.Question)
	return reply
}

func Unmarshal(msg *dns.Msg, resp *Response, udpSize uint16, ednsClientNetmask uint8) *dns.Msg {
	now := time.Now().UTC()

	reply := msg.Copy()
	reply.Truncated = resp.TC
	reply.RecursionDesired = resp.RD
	reply.RecursionAvailable = resp.RA
	reply.AuthenticatedData = resp.AD
	reply.CheckingDisabled = resp.CD
	reply.Rcode = dns.RcodeServerFailure

	reply.Answer = make([]dns.RR, 0, len(resp.Answer))
	for _, rr := range resp.Answer {
		dnsRR, err := unmarshalRR(rr, now)
		if err != nil {
			log.Println(err)
		} else {
			reply.Answer = append(reply.Answer, dnsRR)
		}
	}

	reply.Ns = make([]dns.RR, 0, len(resp.Authority))
	for _, rr := range resp.Authority {
		dnsRR, err := unmarshalRR(rr, now)
		if err != nil {
			log.Println(err)
		} else {
			reply.Ns = append(reply.Ns, dnsRR)
		}
	}

	reply.Extra = make([]dns.RR, 0, len(resp.Additional) + 1)
	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	if udpSize >= 512 {
		opt.SetUDPSize(udpSize)
	} else {
		opt.SetUDPSize(512)
	}
	opt.SetDo(false)
	ednsClientSubnet := resp.EdnsClientSubnet
	ednsClientFamily := uint16(0)
	ednsClientAddress := net.IP(nil)
	ednsClientScope := uint8(255)
	if ednsClientSubnet != "" {
		slash := strings.IndexByte(ednsClientSubnet, '/')
		if slash < 0 {
			log.Println(UnmarshalError { "Invalid client subnet" })
		} else {
			ednsClientAddress = net.ParseIP(ednsClientSubnet[:slash])
			if ednsClientAddress == nil {
				log.Println(UnmarshalError { "Invalid client subnet address" })
			} else if ipv4 := ednsClientAddress.To4(); ipv4 != nil {
				ednsClientFamily = 1
				ednsClientAddress = ipv4
			} else {
				ednsClientFamily = 2
			}
			scope, err := strconv.ParseUint(ednsClientSubnet[slash + 1:], 10, 8)
			if err != nil {
				log.Println(UnmarshalError { "Invalid client subnet address" })
			} else {
				ednsClientScope = uint8(scope)
			}
		}
	}
	if ednsClientAddress != nil {
		if ednsClientNetmask == 255 {
			if ednsClientFamily == 1 {
				ednsClientNetmask = 24
			} else {
				ednsClientNetmask = 48
			}
		}
		edns0Subnet := new(dns.EDNS0_SUBNET)
        edns0Subnet.Code = dns.EDNS0SUBNET
        edns0Subnet.Family = ednsClientFamily
        edns0Subnet.SourceNetmask = ednsClientNetmask
        edns0Subnet.SourceScope = ednsClientScope
        edns0Subnet.Address = ednsClientAddress
        opt.Option = append(opt.Option, edns0Subnet)
	}
	reply.Extra = append(reply.Extra, opt)
	for _, rr := range resp.Additional {
		dnsRR, err := unmarshalRR(rr, now)
		if err != nil {
			log.Println(err)
		} else {
			reply.Extra = append(reply.Extra, dnsRR)
		}
	}

	reply.Rcode = int(resp.Status & 0xf)
	opt.Hdr.Ttl = (opt.Hdr.Ttl & 0x00ffffff) | ((resp.Status & 0xff0) << 20)
	reply.Extra[0] = opt
	return reply
}

func unmarshalRR(rr RR, now time.Time) (dnsRR dns.RR, err error) {
	if strings.ContainsAny(rr.Name, "\t\r\n \"();\\") {
		return nil, UnmarshalError { fmt.Sprintf("Record name contains space: %q", rr.Name) }
	}
	if rr.ExpiresStr != "" {
		rr.Expires, err = time.Parse(time.RFC1123, rr.ExpiresStr)
		if err != nil {
			return nil, UnmarshalError { fmt.Sprintf("Invalid expire time: %q", rr.ExpiresStr) }
		}
		ttl := rr.Expires.Sub(now) / time.Second
		if ttl >= 0 && ttl <= 0xffffffff {
			rr.TTL = uint32(ttl)
		}
	}
	rrType, ok := dns.TypeToString[rr.Type]
	if !ok {
		return nil, UnmarshalError { fmt.Sprintf("Unknown record type: %d", rr.Type) }
	}
	if strings.ContainsAny(rr.Data, "\r\n") {
		return nil, UnmarshalError { fmt.Sprintf("Record data contains newline: %q", rr.Data) }
	}
	zone := fmt.Sprintf("%s %d IN %s %s", rr.Name, rr.TTL, rrType, rr.Data)
	dnsRR, err = dns.NewRR(zone)
	return
}

type UnmarshalError struct {
	err		string
}

func (e UnmarshalError) Error() string {
	return "json-dns: " + e.err
}
