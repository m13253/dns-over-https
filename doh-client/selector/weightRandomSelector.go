package selector

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"sort"
	"sync/atomic"
	"time"
)

type upstreamsInfo struct {
	upstreams []Upstream
	maxWeight int
}

type WeightRandomSelector struct {
	upstreams atomic.Value // upstreamsInfo
	client    http.Client  // http client to check the upstream
}

func NewWeightRandomSelector() *WeightRandomSelector {
	selector := new(WeightRandomSelector)
	selector.upstreams.Store(upstreamsInfo{})

	return selector
}

func (ws *WeightRandomSelector) Add(url string, upstreamType UpstreamType, weight int) (err error) {
	upstreams := ws.upstreams.Load().(upstreamsInfo)

	switch upstreamType {
	case Google:
		upstreams.upstreams = append(upstreams.upstreams, Upstream{
			Type:          Google,
			Url:           url,
			RequestType:   "application/dns-json",
			Weight:        weight,
			CurrentWeight: weight,
		})

	case IETF:
		upstreams.upstreams = append(upstreams.upstreams, Upstream{
			Type:          IETF,
			Url:           url,
			RequestType:   "application/dns-message",
			Weight:        weight,
			CurrentWeight: weight,
		})

	default:
		return errors.New("unknown upstream type")
	}

	upstreams.maxWeight += max(upstreams.maxWeight, weight)

	ws.upstreams.Store(upstreams)
	return nil
}

func (ws *WeightRandomSelector) Evaluate() {
	for {
		originUpstreamsInfo := ws.upstreams.Load().(upstreamsInfo)
		upstreamsInfo := upstreamsInfo{upstreams: make([]Upstream, 0, len(originUpstreamsInfo.upstreams))}

		for _, upstream := range originUpstreamsInfo.upstreams {
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
				upstream.CurrentWeight -= 5
				if upstream.CurrentWeight < 0 {
					upstream.CurrentWeight = 0
				}
				upstreamsInfo.upstreams = append(upstreamsInfo.upstreams, upstream)
				upstreamsInfo.maxWeight = max(upstreamsInfo.maxWeight, upstream.CurrentWeight)
				continue
			}

			switch upstream.Type {
			case Google:
				checkGoogleResponse(resp, &upstream)

			case IETF:
				checkIETFResponse(resp, &upstream)
			}

			upstreamsInfo.upstreams = append(upstreamsInfo.upstreams, upstream)

			cancel()
		}

		sort.Slice(upstreamsInfo, func(i, j int) bool {
			return upstreamsInfo.upstreams[i].CurrentWeight > upstreamsInfo.upstreams[j].CurrentWeight
		})

		ws.upstreams.Store(upstreamsInfo)

		time.Sleep(30 * time.Second)
	}
}

func (ws *WeightRandomSelector) Get() Upstream {
	upstreamsInfo := ws.upstreams.Load().(upstreamsInfo)

	choose := rand.Intn(upstreamsInfo.maxWeight + 1)

	for i := 1; i < len(upstreamsInfo.upstreams); i++ {
		if upstreamsInfo.upstreams[i-1].CurrentWeight >= choose && choose > upstreamsInfo.upstreams[i].CurrentWeight {
			return upstreamsInfo.upstreams[i-1]
		}
	}

	panic("it should not happened, if happened, we should check codes")
}

func checkGoogleResponse(resp *http.Response, upstream *Upstream) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// server error
		upstream.CurrentWeight -= 10
		if upstream.CurrentWeight < 0 {
			upstream.CurrentWeight = 0
		}
		return
	}

	m := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		// should I check error in detail?
		upstream.CurrentWeight -= 1
		if upstream.CurrentWeight < 0 {
			upstream.CurrentWeight = 0
		}
		return
	}

	if status, ok := m["status"]; ok {
		if statusNum, ok := status.(int); ok && statusNum == 0 {
			upstream.CurrentWeight += 5
			if upstream.CurrentWeight > upstream.Weight {
				upstream.CurrentWeight = upstream.Weight
				return
			}
		}
	}

	// should I check error in detail?
	upstream.CurrentWeight -= 1
	if upstream.CurrentWeight < 0 {
		upstream.CurrentWeight = 0
	}
}

func checkIETFResponse(resp *http.Response, upstream *Upstream) {
	defer resp.Body.Close()

	switch resp.StatusCode / 100 {
	case 5:
		// server error
		upstream.CurrentWeight -= 10
		if upstream.CurrentWeight < 0 {
			upstream.CurrentWeight = 0
		}

	case 2:
		upstream.CurrentWeight += 5
		if upstream.CurrentWeight > upstream.Weight {
			upstream.CurrentWeight = upstream.Weight
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
