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

	"github.com/BurntSushi/toml"
)

type config struct {
	Listen         []string `toml:"listen"`
	UpstreamGoogle []string `toml:"upstream_google"`
	UpstreamIETF   []string `toml:"upstream_ietf"`
	Bootstrap      []string `toml:"bootstrap"`
	Timeout        uint     `toml:"timeout"`
	NoCookies      bool     `toml:"no_cookies"`
	NoECS          bool     `toml:"no_ecs"`
	NoIPv6         bool     `toml:"no_ipv6"`
	Verbose        bool     `toml:"verbose"`
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
		conf.Listen = []string{"127.0.0.1:53", "[::1]:53"}
	}
	if len(conf.UpstreamGoogle) == 0 && len(conf.UpstreamIETF) == 0 {
		conf.UpstreamGoogle = []string{"https://dns.google.com/resolve"}
	}
	if conf.Timeout == 0 {
		conf.Timeout = 10
	}

	return conf, nil
}

type configError struct {
	err string
}

func (e *configError) Error() string {
	return e.err
}
