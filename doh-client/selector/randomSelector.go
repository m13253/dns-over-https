package selector

import (
	"errors"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type RandomSelector struct {
	upstreams []*Upstream
}

func NewRandomSelector() *RandomSelector {
	return new(RandomSelector)
}

func (rs *RandomSelector) Add(url string, upstreamType UpstreamType) (err error) {
	switch upstreamType {
	case Google:
		rs.upstreams = append(rs.upstreams, &Upstream{
			Type:        Google,
			Url:         url,
			RequestType: "application/dns-json",
		})

	case IETF:
		rs.upstreams = append(rs.upstreams, &Upstream{
			Type:        IETF,
			Url:         url,
			RequestType: "application/dns-message",
		})

	default:
		return errors.New("unknown upstream type")
	}

	return nil
}

func (rs *RandomSelector) Get() *Upstream {
	return rs.upstreams[rand.Intn(len(rs.upstreams)-1)]
}

func (rs *RandomSelector) StartEvaluate() {}

func (rs *RandomSelector) ReportUpstreamError(upstream *Upstream, upstreamErr upstreamError) {}
