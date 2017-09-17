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
	"time"
)

type Response struct {
	// Standard DNS response code (32 bit integer)
	Status				uint32		`json:"Status"`
	// Whether the response is truncated
	TC					bool		`json:"TC"`
	// Recursion desired
	RD					bool		`json:"RD"`
	// Recursion available
	RA					bool		`json:"RA"`
	// Whether all response data was validated with DNSSEC
	// FIXME: We don't have DNSSEC yet! This bit is not reliable!
	AD					bool		`json:"AD"`
	// Whether the client asked to disable DNSSEC
	CD					bool		`json:"CD"`
	Question			[]Question	`json:"Question"`
	Answer				[]RR		`json:"Answer,omitempty"`
	Authority			[]RR		`json:"Authority,omitempty"`
	Additional			[]RR		`json:"Additional,omitempty"`
	Comment				string		`json:"Comment,omitempty"`
	EdnsClientSubnet	string		`json:"edns_client_subnet,omitempty"`
	// Least time-to-live
	HaveTTL				bool		`json:"-"`
	LeastTTL			uint32		`json:"-"`
	EarliestExpires		time.Time	`json:"-"`
}

type Question struct {
	// FQDN with trailing dot
	Name				string		`json:"name"`
	// Standard DNS RR type
	Type				uint16		`json:"type"`
}

type RR struct {
	Question
	// Record's time-to-live in seconds
	TTL					uint32		`json:"TTL"`
	// TTL in absolute time
	Expires				time.Time	`json:"-"`
	ExpiresStr			string		`json:"Expires"`
	// Data
	Data				string		`json:"data"`
}
