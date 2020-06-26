package main

import (
	"testing"

	"github.com/miekg/dns"
)

func disTestCacheKey(t *testing.T) {
	var s Server

	var req DNSRequest
	req.request = &dns.Msg{}

	s.getCached(&req)
}
