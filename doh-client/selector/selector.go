package selector

type Selector interface {
	// Get returns a upstream
	Get() *Upstream

	// StartEvaluate start upstream evaluation loop
	StartEvaluate()

	// ReportUpstreamError report upstream error
	ReportUpstreamError(upstream *Upstream, upstreamErr upstreamError)
}
