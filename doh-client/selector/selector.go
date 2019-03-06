package selector

type Selector interface {
	Get() Upstream
	Evaluate()
}
