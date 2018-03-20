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
	"time"
)

type Response struct {
	// Standard DNS response code (32 bit integer)
	Status uint32 `json:"Status"`
	// Whether the response is truncated
	TC bool `json:"TC"`
	// Recursion desired
	RD bool `json:"RD"`
	// Recursion available
	RA bool `json:"RA"`
	// Whether all response data was validated with DNSSEC
	// FIXME: We don't have DNSSEC yet! This bit is not reliable!
	AD bool `json:"AD"`
	// Whether the client asked to disable DNSSEC
	CD               bool       `json:"CD"`
	Question         []Question `json:"Question"`
	Answer           []RR       `json:"Answer,omitempty"`
	Authority        []RR       `json:"Authority,omitempty"`
	Additional       []RR       `json:"Additional,omitempty"`
	Comment          string     `json:"Comment,omitempty"`
	EdnsClientSubnet string     `json:"edns_client_subnet,omitempty"`
	// Least time-to-live
	HaveTTL         bool      `json:"-"`
	LeastTTL        uint32    `json:"-"`
	EarliestExpires time.Time `json:"-"`
}

type Question struct {
	// FQDN with trailing dot
	Name string `json:"name"`
	// Standard DNS RR type
	Type uint16 `json:"type"`
}

type RR struct {
	Question
	// Record's time-to-live in seconds
	TTL uint32 `json:"TTL"`
	// TTL in absolute time
	Expires    time.Time `json:"-"`
	ExpiresStr string    `json:"Expires"`
	// Data
	Data string `json:"data"`
}
