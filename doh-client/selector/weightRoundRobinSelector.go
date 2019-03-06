package selector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"time"
)

type WeightRoundRobinSelector struct {
	upstreams atomic.Value // upstreamsInfo
	client    http.Client  // http client to check the upstream
}

func NewWeightRoundRobinSelector() *WeightRoundRobinSelector {
	selector := new(WeightRoundRobinSelector)
	selector.upstreams.Store(make([]Upstream, 0))

	return selector
}

func (ws *WeightRoundRobinSelector) Add(url string, upstreamType UpstreamType, weight int) (err error) {
	upstreams := ws.upstreams.Load().([]Upstream)

	switch upstreamType {
	case Google:
		upstreams = append(upstreams, Upstream{
			Type:            Google,
			Url:             url,
			RequestType:     "application/dns-json",
			weight:          weight,
			effectiveWeight: weight,
		})

	case IETF:
		upstreams = append(upstreams, Upstream{
			Type:            IETF,
			Url:             url,
			RequestType:     "application/dns-message",
			weight:          weight,
			effectiveWeight: weight,
		})

	default:
		return errors.New("unknown upstream type")
	}

	ws.upstreams.Store(upstreams)
	return nil
}

// COW, avoid concurrent read write upstreams
func (ws *WeightRoundRobinSelector) Evaluate() {
	for {
		originUpstreams := ws.upstreams.Load().([]Upstream)
		upstreams := make([]Upstream, 0, len(originUpstreams))

		for _, upstream := range originUpstreams {
			upstreamUrl := upstream.Url
			var acceptType string

			switch upstream.Type {
			case Google:
				upstreamUrl += "?name=www.example.com&type=A"
				acceptType = "application/dns-json"

			case IETF:
				// www.example.com
				upstreamUrl += "?dns=q80BAAABAAAAAAAAA3d3dwdleGFtcGxlA2NvbQAAAQAB"
				acceptType = "application/dns-message"
			}

			req, err := http.NewRequest(http.MethodGet, upstreamUrl, nil)
			if err != nil {
				/*log.Println("upstream:", upstreamUrl, "type:", typeMap[upstream.Type], "check failed:", err)
				continue*/

				// should I only log it? But if there is an error, I think when query the server will return error too
				panic("upstream: " + upstreamUrl + " type: " + typeMap[upstream.Type] + " check failed: " + err.Error())
			}

			req.Header.Set("accept", acceptType)

			timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)

			req = req.WithContext(timeout)

			resp, err := ws.client.Do(req)
			if err != nil {
				// should I check error in detail?
				upstream.effectiveWeight -= 5
				if upstream.effectiveWeight < 0 {
					upstream.effectiveWeight = 0
				}
				upstreams = append(upstreams, upstream)
				continue
			}

			switch upstream.Type {
			case Google:
				checkGoogleResponse(resp, &upstream)

			case IETF:
				checkIETFResponse(resp, &upstream)
			}

			upstreams = append(upstreams, upstream)

			cancel()
		}

		ws.upstreams.Store(upstreams)

		time.Sleep(30 * time.Second)
	}
}

// nginx wrr like
func (ws *WeightRoundRobinSelector) Get() Upstream {
	var (
		total             int
		bestUpstreamIndex = -1
	)

	upstreams := ws.upstreams.Load().([]Upstream)

	for i := range upstreams {
		upstreams[i].currentWeight += upstreams[i].effectiveWeight
		total += upstreams[i].effectiveWeight

		if bestUpstreamIndex == -1 || upstreams[i].currentWeight > upstreams[bestUpstreamIndex].currentWeight {
			bestUpstreamIndex = i
		}
	}

	upstreams[bestUpstreamIndex].currentWeight -= total

	return upstreams[bestUpstreamIndex]
}

func checkGoogleResponse(resp *http.Response, upstream *Upstream) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// server error
		upstream.effectiveWeight -= 10
		if upstream.effectiveWeight < 0 {
			upstream.effectiveWeight = 0
		}
		return
	}

	m := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		// should I check error in detail?
		upstream.effectiveWeight -= 1
		if upstream.effectiveWeight < 0 {
			upstream.effectiveWeight = 0
		}
		return
	}

	if status, ok := m["status"]; ok {
		if statusNum, ok := status.(int); ok && statusNum == 0 {
			upstream.effectiveWeight += 5
			if upstream.effectiveWeight > upstream.weight {
				upstream.effectiveWeight = upstream.weight
				return
			}
		}
	}

	// should I check error in detail?
	upstream.effectiveWeight -= 1
	if upstream.effectiveWeight < 0 {
		upstream.effectiveWeight = 0
	}
}

func checkIETFResponse(resp *http.Response, upstream *Upstream) {
	defer resp.Body.Close()

	switch resp.StatusCode / 100 {
	case 5:
		// server error
		upstream.effectiveWeight -= 10
		if upstream.effectiveWeight < 0 {
			upstream.effectiveWeight = 0
		}

	case 2:
		upstream.effectiveWeight += 5
		if upstream.effectiveWeight > upstream.weight {
			upstream.effectiveWeight = upstream.weight
		}

		// TODO anything else?
	}
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}
