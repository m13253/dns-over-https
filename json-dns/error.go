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
	"encoding/json"
	"log"
	"net/http"
	"github.com/miekg/dns"
)

type dnsError struct {
	Status				uint32		`json:"Status"`
	Comment				string		`json:"Comment,omitempty"`
}

func FormatError(w http.ResponseWriter, comment string, errcode int) {
	errJson := dnsError {
		Status: dns.RcodeServerFailure,
		Comment: comment,
	}
	errStr, err := json.Marshal(errJson)
	if err != nil {
		log.Fatalln(err)
	}
	w.WriteHeader(errcode)
	w.Write(errStr)
}
