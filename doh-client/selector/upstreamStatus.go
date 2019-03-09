package selector

type upstreamStatus int

const (
	// when query upstream timeout, usually upstream is unavailable for a long time
	Timeout upstreamStatus = iota

	// when query upstream return 5xx response, upstream still alive, maybe just a lof of query for him
	Error

	// when query upstream ok, means upstream is available
	OK
)
