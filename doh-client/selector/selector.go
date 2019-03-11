package selector

type Selector interface {
	// Get returns a upstream
	Get() *Upstream

	// StartEvaluate start upstream evaluation loop
	StartEvaluate()

	// ReportUpstreamStatus report upstream status
	ReportUpstreamStatus(upstream *Upstream, upstreamStatus upstreamStatus)
}

type DebugReporter interface {
	// ReportWeights starts a goroutine to report all upstream weights, recommend interval is 15s
	ReportWeights()
}
