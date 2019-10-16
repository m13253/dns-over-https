package main

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"time"

	"github.com/miekg/dns"
)

const debugCache = false

func getRequestKey(msg *dns.Msg) (uint64, string) {
	// blank out ID before creating hash
	id := msg.Id
	msg.Id = 0

	// take away extra options, e.g. TSIG and EDNS
	extras := msg.Extra
	msg.Extra = nil

	// use JSON to get data representation
	var b []byte
	var err error
	if !debugCache {
		b, err = json.Marshal(msg)
	} else {
		b, err = json.MarshalIndent(msg, "", "\t")
	}
	if err != nil {
		panic(err)
	}
	msg.Id = id
	msg.Extra = extras

	// calculate hash
	h := fnv.New64a()
	h.Write(b)

	if debugCache {
		return h.Sum64(), string(b)
	} else {
		return h.Sum64(), ""
	}
}

// getCached will set req.response and return true in case of cache hit;
// return false otherwise.
func (s *Server) getCached(req *DNSRequest) bool {
	if !s.conf.Caching {
		return false
	}

	req.key, req.keyJSON = getRequestKey(req.request)
	if !debugCache {
		req.keyJSON = ""
	}

	s.cacheLock.RLock()
	e, ok := s.cache[req.key]
	s.cacheLock.RUnlock()

	if ok {
		log.Printf("cache hit for request 0x%x\n", req.key)
		req.currentUpstream = e.upstream
		req.response = &e.Msg

		// copy request ID to response ID
		req.response.Id = req.request.Id
	}

	return ok
}

func (s *Server) saveCached(req *DNSRequest) {
	if !s.conf.Caching {
		return
	}

	// remove extras like TSIG and EDNS
	//TODO: check if this is always safe
	req.response.Extra = nil

	s.cacheLock.Lock()

	// remove any entry which contains expired data
	now := time.Now()
	for k, e := range s.cache {
		if e.expiresAt.Before(now) {
			delete(s.cache, k)
		}
	}

	ttl := findMinimumTTL(req.response)
	if ttl == 0 {
		// no caching
		s.cacheLock.Unlock()
		return
	}

	var e cacheEntry
	e.upstream = req.currentUpstream
	// copy response
	req.response.CopyTo(&e.Msg)
	// set expiration time
	e.expiresAt = time.Now().Add(ttl)

	// store new entry
	s.cache[req.key] = e

	s.cacheLock.Unlock()

	if debugCache {
		b, err := json.MarshalIndent(req.response, "", "\t")
		if err != nil {
			panic(err)
		}
		log.Printf("cache miss for request 0x%x (TTL=%v)\nREQUEST:\n%s\nRESPONSE:\n%s\n", req.key, ttl.Round(time.Second), req.keyJSON, string(b))
	} else {
		log.Printf("cache miss for request 0x%x (TTL=%v)\n", req.key, ttl.Round(time.Second))
	}
}

const maxTTLDuration = time.Duration(2147483647) * time.Second

func findMinimumTTL(msg *dns.Msg) time.Duration {
	shortestTTL := maxTTLDuration

	for _, a := range msg.Answer {
		hdr := a.Header()
		// re-cast until upstream sign bug is fixed
		// https://github.com/miekg/dns/issues/956
		ttl := int32(hdr.Ttl)
		if ttl <= 0 {
			ttl = 0
		}

		if ttl == 0 {
			return 0
		}

		d := time.Duration(ttl) * time.Second
		if d < shortestTTL {
			shortestTTL = d
		}
	}

	// no answer record found, use a default TTL
	if maxTTLDuration == shortestTTL {
		shortestTTL = time.Second * 30
	}

	return shortestTTL
}
