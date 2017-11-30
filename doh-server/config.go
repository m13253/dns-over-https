/*
    DNS-over-HTTPS
    Copyright (C) 2017 Star Brilliant <m13253@hotmail.com>

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
	Listen		string		`toml:"listen"`
	Cert		string		`toml:"cert"`
	Key			string		`toml:"key"`
	Path		string		`toml:"path"`
	Upstream	[]string	`toml:"upstream"`
	Tries		uint		`toml:"tries"`
	TCPOnly		bool		`toml:"tcp_only"`
	Verbose		bool		`toml:"verbose"`
}

func loadConfig(path string) (*config, error) {
	conf := &config {}
	metaData, err := toml.DecodeFile(path, conf)
	if err != nil {
		return nil, err
	}
	for _, key := range metaData.Undecoded() {
		return nil, &configError { fmt.Sprintf("unknown option %q", key.String()) }
	}

	if conf.Listen == "" {
		conf.Listen = "127.0.0.1:8053"
	}
	if conf.Path == "" {
		conf.Path = "/resolve"
	}
	if len(conf.Upstream) == 0 {
		conf.Upstream = []string { "8.8.8.8:53", "8.8.4.4:53" }
	}
	if conf.Tries == 0 {
		conf.Tries = 3
	}

	if (conf.Cert != "") != (conf.Key != "") {
		return nil, &configError { "You must specify both -cert and -key to enable TLS" }
	}

	return conf, nil
}

type configError struct {
	err		string
}

func (e *configError) Error() string {
	return e.err
}
