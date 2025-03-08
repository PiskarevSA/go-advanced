package usecases

type Repositories interface {
	SetGauge(key string, value float64)
	Gauge(key string) (value float64, exist bool)
	SetCounter(key string, value int64)
	Counter(key string) (value int64, exist bool)
	Dump() (gauge map[string]float64, counter map[string]int64)
}

// metrics usecase is just a wrapper to repo for now
type Metrics struct {
	repo Repositories
}

func NewMetrics(repo Repositories) *Metrics {
	return &Metrics{
		repo: repo,
	}
}

func (m *Metrics) SetGauge(key string, value float64) {
	m.repo.SetGauge(key, value)
}

func (m *Metrics) Gauge(key string) (value float64, exist bool) {
	return m.repo.Gauge(key)
}

func (m *Metrics) SetCounter(key string, value int64) {
	m.repo.SetCounter(key, value)
}

func (m *Metrics) Counter(key string) (value int64, exist bool) {
	return m.repo.Counter(key)
}

func (m *Metrics) Dump() (gauge map[string]float64, counter map[string]int64) {
	return m.repo.Dump()
}
