package selector

type Selector interface {
	// Get returns a upstream
	Get() *Upstream

	// StartEvaluate start upstream evaluation loop
	StartEvaluate()

	// ReportUpstreamStatus report upstream status
	ReportUpstreamStatus(upstream *Upstream, upstreamStatus upstreamStatus)
}
