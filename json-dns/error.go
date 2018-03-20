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
	"encoding/json"
	"log"
	"net/http"

	"github.com/miekg/dns"
)

type dnsError struct {
	Status  uint32 `json:"Status"`
	Comment string `json:"Comment,omitempty"`
}

func FormatError(w http.ResponseWriter, comment string, errcode int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	errJson := dnsError{
		Status:  dns.RcodeServerFailure,
		Comment: comment,
	}
	errStr, err := json.Marshal(errJson)
	if err != nil {
		log.Fatalln(err)
	}
	w.WriteHeader(errcode)
	w.Write(errStr)
}
