package selector

type upstreamError int

const (
	// when query upstream timeout, usually upstream is unavailable for a long time
	Serious upstreamError = iota

	// when query upstream return 5xx response, upstream still alive, maybe just a lof of query for him
	Medium
)
