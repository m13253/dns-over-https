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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
)

func checkPIDFile(pidFile string) (bool, error) {
retry:
	f, err := os.OpenFile(pidFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if os.IsExist(err) {
		pidStr, err := ioutil.ReadFile(pidFile)
		if err != nil {
			return false, err
		}
		pid, err := strconv.ParseUint(string(pidStr), 10, 0)
		if err != nil {
			return false, err
		}
		_, err = os.Stat(fmt.Sprintf("/proc/%d", pid))
		if os.IsNotExist(err) {
			err = os.Remove(pidFile)
			if err != nil {
				return false, err
			}
			goto retry
		} else if err != nil {
			return false, err
		}
		log.Printf("Already running on PID %d, exiting.\n", pid)
		return false, nil
	} else if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = io.WriteString(f, strconv.FormatInt(int64(os.Getpid()), 10))
	if err != nil {
		return false, err
	}
	return true, nil
}

func main() {
	confPath := flag.String("conf", "doh-server.conf", "Configuration file")
	verbose := flag.Bool("verbose", false, "Enable logging")
	showVersion := flag.Bool("version", false, "Show software version and exit")
	var pidFile *string

	// I really want to push the technology forward by recommending cgroup-based
	// process tracking. But I understand some cloud service providers have
	// their own monitoring system. So this feature is only enabled on Linux and
	// BSD series platforms which lacks functionality similar to cgroup.
	switch runtime.GOOS {
	case "dragonfly", "freebsd", "linux", "netbsd", "openbsd":
		pidFile = flag.String("pid-file", "", "PID file for legacy supervision systems lacking support for reliable cgroup-based process tracking")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("doh-server %s\nHomepage: https://github.com/m13253/dns-over-https\n", VERSION)
		return
	}

	if pidFile != nil && *pidFile != "" {
		ok, err := checkPIDFile(*pidFile)
		if err != nil {
			log.Printf("Error checking PID file: %v\n", err)
		}
		if !ok {
			return
		}
	}

	conf, err := loadConfig(*confPath)
	if err != nil {
		log.Fatalln(err)
	}

	if *verbose {
		conf.Verbose = true
	}

	server, err := NewServer(conf)
	if err != nil {
		log.Fatalln(err)
	}
	_ = server.Start()
}
