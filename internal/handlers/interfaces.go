package handlers

type Repositories interface {
	SetGauge(key string, value float64)
	Gauge(key string) (value float64, exist bool)
	IncreaseCounter(key string, addition int64)
	Counter(key string) (value int64, exist bool)
}
