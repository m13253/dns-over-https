package selector

import "fmt"

type UpstreamType int

const (
	Google UpstreamType = iota
	IETF
)

var typeMap = map[UpstreamType]string{
	Google: "Google",
	IETF:   "IETF",
}

type Upstream struct {
	Type            UpstreamType
	Url             string
	RequestType     string
	weight          int
	effectiveWeight int
	currentWeight   int
}

func (u Upstream) String() string {
	return fmt.Sprintf("upstream type: %s, upstream url: %s", typeMap[u.Type], u.Url)
}
