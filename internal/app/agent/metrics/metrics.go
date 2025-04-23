package metrics

type (
	Gauge   float64
	Counter int64
)

type Metrics struct {
	Gauge   map[string]Gauge
	Counter map[string]Counter
}

func NewMetrics() *Metrics {
	return &Metrics{
		Gauge:   make(map[string]Gauge, 0),
		Counter: make(map[string]Counter, 0),
	}
}
