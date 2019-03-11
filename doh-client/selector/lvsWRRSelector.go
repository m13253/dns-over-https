package selector

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LVSWRRSelector struct {
	upstreams     []*Upstream // upstreamsInfo
	client        http.Client // http client to check the upstream
	lastChoose    int32
	currentWeight int32
}

func NewLVSWRRSelector(timeout time.Duration) *LVSWRRSelector {
	return &LVSWRRSelector{
		client:     http.Client{Timeout: timeout},
		lastChoose: -1,
	}
}

func (ls *LVSWRRSelector) Add(url string, upstreamType UpstreamType, weight int32) (err error) {
	if weight < 1 {
		return errors.New("weight is 1")
	}

	switch upstreamType {
	case Google:
		ls.upstreams = append(ls.upstreams, &Upstream{
			Type:            Google,
			URL:             url,
			RequestType:     "application/dns-json",
			weight:          weight,
			effectiveWeight: weight,
		})

	case IETF:
		ls.upstreams = append(ls.upstreams, &Upstream{
			Type:            IETF,
			URL:             url,
			RequestType:     "application/dns-message",
			weight:          weight,
			effectiveWeight: weight,
		})

	default:
		return errors.New("unknown upstream type")
	}

	return nil
}

func (ls *LVSWRRSelector) StartEvaluate() {
	go func() {
		for {
			wg := sync.WaitGroup{}

			for i := range ls.upstreams {
				wg.Add(1)

				go func(i int) {
					defer wg.Done()

					upstreamURL := ls.upstreams[i].URL
					var acceptType string

					switch ls.upstreams[i].Type {
					case Google:
						upstreamURL += "?name=www.example.com&type=A"
						acceptType = "application/dns-json"

					case IETF:
						// www.example.com
						upstreamURL += "?dns=q80BAAABAAAAAAAAA3d3dwdleGFtcGxlA2NvbQAAAQAB"
						acceptType = "application/dns-message"
					}

					req, err := http.NewRequest(http.MethodGet, upstreamURL, nil)
					if err != nil {
						/*log.Println("upstream:", upstreamURL, "type:", typeMap[upstream.Type], "check failed:", err)
						continue*/

						// should I only log it? But if there is an error, I think when query the server will return error too
						panic("upstream: " + upstreamURL + " type: " + typeMap[ls.upstreams[i].Type] + " check failed: " + err.Error())
					}

					req.Header.Set("accept", acceptType)

					resp, err := ls.client.Do(req)
					if err != nil {
						// should I check error in detail?
						if atomic.AddInt32(&ls.upstreams[i].effectiveWeight, -5) < 1 {
							atomic.StoreInt32(&ls.upstreams[i].effectiveWeight, 1)
						}
						return
					}

					switch ls.upstreams[i].Type {
					case Google:
						ls.checkGoogleResponse(resp, ls.upstreams[i])

					case IETF:
						ls.checkIETFResponse(resp, ls.upstreams[i])
					}
				}(i)
			}

			wg.Wait()

			time.Sleep(15 * time.Second)
		}
	}()
}

func (ls *LVSWRRSelector) Get() *Upstream {
	if len(ls.upstreams) == 1 {
		return ls.upstreams[0]
	}

	for {
		atomic.StoreInt32(&ls.lastChoose, (atomic.LoadInt32(&ls.lastChoose)+1)%int32(len(ls.upstreams)))

		if atomic.LoadInt32(&ls.lastChoose) == 0 {
			atomic.AddInt32(&ls.currentWeight, -ls.gcdWeight())

			if atomic.LoadInt32(&ls.currentWeight) <= 0 {
				atomic.AddInt32(&ls.currentWeight, ls.maxWeight())

				if atomic.LoadInt32(&ls.currentWeight) == 0 {
					panic("current weight is 0")
				}
			}
		}

		if atomic.LoadInt32(&ls.upstreams[atomic.LoadInt32(&ls.lastChoose)].effectiveWeight) >= atomic.LoadInt32(&ls.currentWeight) {
			return ls.upstreams[atomic.LoadInt32(&ls.lastChoose)]
		}
	}
}

func (ls *LVSWRRSelector) gcdWeight() (res int32) {
	res = gcd(atomic.LoadInt32(&ls.upstreams[0].effectiveWeight), atomic.LoadInt32(&ls.upstreams[0].effectiveWeight))

	for i := 1; i < len(ls.upstreams); i++ {
		res = gcd(res, atomic.LoadInt32(&ls.upstreams[i].effectiveWeight))
	}

	return
}

func (ls *LVSWRRSelector) maxWeight() (res int32) {
	for _, upstream := range ls.upstreams {
		w := atomic.LoadInt32(&upstream.effectiveWeight)
		if w > res {
			res = w
		}
	}

	return
}

func gcd(x, y int32) int32 {
	for {
		if x < y {
			x, y = y, x
		}

		tmp := x % y
		if tmp == 0 {
			return y
		}

		x = tmp
	}
}

func (ls *LVSWRRSelector) ReportUpstreamStatus(upstream *Upstream, upstreamStatus upstreamStatus) {
	switch upstreamStatus {
	case Timeout:
		if atomic.AddInt32(&upstream.effectiveWeight, -5) < 1 {
			atomic.StoreInt32(&upstream.effectiveWeight, 1)
		}

	case Error:
		if atomic.AddInt32(&upstream.effectiveWeight, -2) < 1 {
			atomic.StoreInt32(&upstream.effectiveWeight, 1)
		}

	case OK:
		if atomic.AddInt32(&upstream.effectiveWeight, 1) > upstream.weight {
			atomic.StoreInt32(&upstream.effectiveWeight, upstream.weight)
		}
	}
}

func (ls *LVSWRRSelector) checkGoogleResponse(resp *http.Response, upstream *Upstream) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// server error
		if atomic.AddInt32(&upstream.effectiveWeight, -3) < 1 {
			atomic.StoreInt32(&upstream.effectiveWeight, 1)
		}
		return
	}

	m := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		// should I check error in detail?
		if atomic.AddInt32(&upstream.effectiveWeight, -2) < 1 {
			atomic.StoreInt32(&upstream.effectiveWeight, 1)
		}
		return
	}

	if status, ok := m["Status"]; ok {
		if statusNum, ok := status.(float64); ok && statusNum == 0 {
			if atomic.AddInt32(&upstream.effectiveWeight, 5) > upstream.weight {
				atomic.StoreInt32(&upstream.effectiveWeight, upstream.weight)
			}
			return
		}
	}

	// should I check error in detail?
	if atomic.AddInt32(&upstream.effectiveWeight, -2) < 1 {
		atomic.StoreInt32(&upstream.effectiveWeight, 1)
	}
}

func (ls *LVSWRRSelector) checkIETFResponse(resp *http.Response, upstream *Upstream) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// server error
		if atomic.AddInt32(&upstream.effectiveWeight, -3) < 1 {
			atomic.StoreInt32(&upstream.effectiveWeight, 1)
		}
		return
	}

	if atomic.AddInt32(&upstream.effectiveWeight, 5) > upstream.weight {
		atomic.StoreInt32(&upstream.effectiveWeight, upstream.weight)
	}
}

func (ls *LVSWRRSelector) ReportWeights() {
	go func() {
		for {
			time.Sleep(15 * time.Second)

			for _, u := range ls.upstreams {
				log.Printf("%s, effect weight: %d", u, atomic.LoadInt32(&u.effectiveWeight))
			}
		}
	}()
}
