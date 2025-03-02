package storage

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (m *MemStorage) SetGauge(key string, value float64) {
	m.gauge[key] = value
}

func (m *MemStorage) Gauge(key string) (value float64, exist bool) {
	value, exist = m.gauge[key]
	return
}

func (m *MemStorage) IncreaseCounter(key string, addition int64) {
	m.counter[key] += addition
}

func (m *MemStorage) Counter(key string) (value int64, exist bool) {
	value, exist = m.counter[key]
	return
}
