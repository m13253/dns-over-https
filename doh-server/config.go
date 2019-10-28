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

package main

import (
	"fmt"
	"regexp"

	"github.com/BurntSushi/toml"
)

type config struct {
	Listen           []string `toml:"listen"`
	LocalAddr        string   `toml:"local_addr"`
	Cert             string   `toml:"cert"`
	Key              string   `toml:"key"`
	Path             string   `toml:"path"`
	Upstream         []string `toml:"upstream"`
	Timeout          uint     `toml:"timeout"`
	Tries            uint     `toml:"tries"`
	Verbose          bool     `toml:"verbose"`
	DebugHTTPHeaders []string `toml:"debug_http_headers"`
	LogGuessedIP     bool     `toml:"log_guessed_client_ip"`
}

func loadConfig(path string) (*config, error) {
	conf := &config{}
	metaData, err := toml.DecodeFile(path, conf)
	if err != nil {
		return nil, err
	}
	for _, key := range metaData.Undecoded() {
		return nil, &configError{fmt.Sprintf("unknown option %q", key.String())}
	}

	if len(conf.Listen) == 0 {
		conf.Listen = []string{"127.0.0.1:8053", "[::1]:8053"}
	}

	if conf.Path == "" {
		conf.Path = "/dns-query"
	}
	if len(conf.Upstream) == 0 {
		conf.Upstream = []string{"udp:8.8.8.8:53", "udp:8.8.4.4:53"}
	}
	if conf.Timeout == 0 {
		conf.Timeout = 10
	}
	if conf.Tries == 0 {
		conf.Tries = 1
	}

	if (conf.Cert != "") != (conf.Key != "") {
		return nil, &configError{"You must specify both -cert and -key to enable TLS"}
	}

	// validate all upstreams
	for _, us := range conf.Upstream {
		address, t := addressAndType(us)
		if address == "" {
			return nil, &configError{"One of the upstreams has not a (udp|tcp|tcp-tls) prefix e.g. udp:1.1.1.1:53"}
		}

		switch t {
		case "tcp", "udp", "tcp-tls":
			// OK
		default:
			return nil, &configError{"Invalid upstream prefix specified, choose one of: udp tcp tcp-tls"}
		}
	}

	return conf, nil
}

var rxUpstreamWithTypePrefix = regexp.MustCompile("^[a-z-]+(:)")

func addressAndType(us string) (string, string) {
	p := rxUpstreamWithTypePrefix.FindStringSubmatchIndex(us)
	if len(p) != 4 {
		return "", ""
	}

	return us[p[2]+1:], us[:p[2]]
}

type configError struct {
	err string
}

func (e *configError) Error() string {
	return e.err
}
