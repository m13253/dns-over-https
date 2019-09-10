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

package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

const (
	Random   = "random"
	NginxWRR = "weighted_round_robin"
	LVSWRR   = "lvs_weighted_round_robin"
)

type upstreamDetail struct {
	URL    string `toml:"url"`
	Weight int32  `toml:"weight"`
}

type upstream struct {
	UpstreamGoogle   []upstreamDetail `toml:"upstream_google"`
	UpstreamIETF     []upstreamDetail `toml:"upstream_ietf"`
	UpstreamSelector string           `toml:"upstream_selector"` // usable: random or weighted_random
}

type others struct {
	Bootstrap        []string `toml:"bootstrap"`
	Passthrough      []string `toml:"passthrough"`
	Timeout          uint     `toml:"timeout"`
	NoCookies        bool     `toml:"no_cookies"`
	NoECS            bool     `toml:"no_ecs"`
	NoIPv6           bool     `toml:"no_ipv6"`
	NoUserAgent      bool     `toml:"no_user_agent"`
	Verbose          bool     `toml:"verbose"`
	DebugHTTPHeaders []string `toml:"debug_http_headers"`
}

type Config struct {
	Listen   []string `toml:"listen"`
	Upstream upstream `toml:"upstream"`
	Other    others   `toml:"others"`
}

func LoadConfig(path string) (*Config, error) {
	conf := &Config{}
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
	if len(conf.Upstream.UpstreamGoogle) == 0 && len(conf.Upstream.UpstreamIETF) == 0 {
		conf.Upstream.UpstreamGoogle = []upstreamDetail{{URL: "https://dns.google.com/resolve", Weight: 50}}
	}
	if conf.Other.Timeout == 0 {
		conf.Other.Timeout = 10
	}

	if conf.Upstream.UpstreamSelector == "" {
		conf.Upstream.UpstreamSelector = Random
	}

	return conf, nil
}

type configError struct {
	err string
}

func (e *configError) Error() string {
	return e.err
}
