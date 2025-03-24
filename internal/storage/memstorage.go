package storage

import (
	"encoding/json"
	"io"
	"sync"
)

type MemStorage struct {
	mutex sync.RWMutex

	GaugeMap   map[string]float64 `json:"gauge"`
	CounterMap map[string]int64   `json:"counter"`
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		GaugeMap:   make(map[string]float64),
		CounterMap: make(map[string]int64),
	}
}

func (m *MemStorage) SetGauge(key string, value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.GaugeMap[key] = value
}

func (m *MemStorage) Gauge(key string) (value float64, exists bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, exists = m.GaugeMap[key]
	return
}

func (m *MemStorage) IncreaseCounter(key string, delta int64) int64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.CounterMap[key] += delta
	return m.CounterMap[key]
}

func (m *MemStorage) Counter(key string) (value int64, exists bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, exists = m.CounterMap[key]
	return
}

func (m *MemStorage) Dump() (gauge map[string]float64, counter map[string]int64) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	gauge = make(map[string]float64)
	for k, v := range m.GaugeMap {
		gauge[k] = v
	}

	counter = make(map[string]int64)
	for k, v := range m.CounterMap {
		counter[k] = v
	}
	return
}

func (m *MemStorage) Load(r io.Reader) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return json.NewDecoder(r).Decode(&m)
}

func (m *MemStorage) Store(w io.Writer) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return json.NewEncoder(w).Encode(m)
}
