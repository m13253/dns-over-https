package selector

import (
	"errors"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type HashSelector struct {
	upstreams []*Upstream
}

func NewHashSelector() *HashSelector {
	return new(HashSelector)
}

func (rs *HashSelector) Add(url string, upstreamType UpstreamType) (err error) {
	switch upstreamType {
	case Google:
		rs.upstreams = append(rs.upstreams, &Upstream{
			Type:        Google,
			URL:         url,
			RequestType: "application/dns-json",
		})

	case IETF:
		rs.upstreams = append(rs.upstreams, &Upstream{
			Type:        IETF,
			URL:         url,
			RequestType: "application/dns-message",
		})

	default:
		return errors.New("unknown upstream type")
	}

	return nil
}

func (rs *HashSelector) Get() *Upstream {
    // here, if we have the name to be resolved (a string)
    // we could compute the modulo over the size of upstream servers
    // something like url.hash()%len(rs.upstreams)
    // how to refactor Get() to get the name
	return rs.upstreams[url.hash()%len(rs.upstreams)]
}

func (rs *HashSelector) StartEvaluate() {}

func (rs *HashSelector) ReportUpstreamStatus(upstream *Upstream, upstreamStatus upstreamStatus) {}
