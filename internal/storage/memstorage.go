package storage

import "sync"

// TODO PR #5
// Вместо sync.Mutex можно использовать sync.RWMutex и соответственно
// sync.RLock sync.RUnlock, которые не блокируется при чтении
type MemStorage struct {
	mutex sync.Mutex

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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.gauge[key] = value
}

func (m *MemStorage) Gauge(key string) (value float64, exist bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	value, exist = m.gauge[key]
	return
}

func (m *MemStorage) SetCounter(key string, value int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// TBD m.counter[key] = value
	// see https://app.pachca.com/chats/19865306?message=445288954
	m.counter[key] += value
}

func (m *MemStorage) Counter(key string) (value int64, exist bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	value, exist = m.counter[key]
	return
}

func (m *MemStorage) Dump() (gauge map[string]float64, counter map[string]int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	gauge = make(map[string]float64)
	for k, v := range m.gauge {
		gauge[k] = v
	}

	counter = make(map[string]int64)
	for k, v := range m.counter {
		counter[k] = v
	}
	return
}
