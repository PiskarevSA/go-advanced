package metrics

import (
	"maps"
	"sync"
	"sync/atomic"
)

type (
	Gauge   float64
	Counter int64
)

type PollFunc func(gauge map[string]Gauge, counter map[string]Counter)

type Metrics struct {
	mutex     sync.RWMutex
	pollFunc  PollFunc
	gauge     map[string]Gauge
	counter   map[string]Counter
	readyRead atomic.Bool
}

func New(pollFunc PollFunc) *Metrics {
	return &Metrics{
		pollFunc: pollFunc,
		gauge:    make(map[string]Gauge),
		counter:  make(map[string]Counter),
	}
}

// returns poll count
func (m *Metrics) Poll() Counter {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.pollFunc(m.gauge, m.counter)

	result := Counter(-1)
	if pollCount, ok := m.counter["PollCount"]; ok {
		result = pollCount
	}
	m.readyRead.Store(true)
	return result
}

func (m *Metrics) Get() (Counter, map[string]Gauge, map[string]Counter) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	gauge := make(map[string]Gauge)
	maps.Copy(gauge, m.gauge)

	counter := make(map[string]Counter)
	maps.Copy(counter, m.counter)

	if pollCount, ok := m.counter["PollCount"]; ok {
		return pollCount, gauge, counter
	}
	return -1, gauge, counter
}

func (m *Metrics) ReadyRead() bool {
	return m.readyRead.Load()
}
