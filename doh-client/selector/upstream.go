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
	URL             string
	RequestType     string
	weight          int32
	effectiveWeight int32
	currentWeight   int32
}

func (u Upstream) String() string {
	return fmt.Sprintf("upstream type: %s, upstream url: %s", typeMap[u.Type], u.URL)
}
