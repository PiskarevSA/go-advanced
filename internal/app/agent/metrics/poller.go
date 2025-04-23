package metrics

import (
	"maps"
	"sync"
	"sync/atomic"
)

type PollFunc func(gauge map[string]Gauge, counter map[string]Counter)

type Poller struct {
	mutex     sync.RWMutex
	pollFunc  PollFunc
	metrics   Metrics
	readyRead atomic.Bool
	pollCount int
}

func NewPoller(pollFunc PollFunc) *Poller {
	return &Poller{
		pollFunc: pollFunc,
		metrics:  *NewMetrics(),
	}
}

// returns poll count
func (m *Poller) Poll() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.pollFunc(m.metrics.Gauge, m.metrics.Counter)

	m.readyRead.Store(true)

	m.pollCount++
	return m.pollCount
}

func (m *Poller) Get() (int, map[string]Gauge, map[string]Counter) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	gauge := make(map[string]Gauge)
	maps.Copy(gauge, m.metrics.Gauge)

	counter := make(map[string]Counter)
	maps.Copy(counter, m.metrics.Counter)

	return m.pollCount, gauge, counter
}

func (m *Poller) ReadyRead() bool {
	return m.readyRead.Load()
}
