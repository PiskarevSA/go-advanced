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
func (p *Poller) Poll() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.pollFunc(p.metrics.Gauge, p.metrics.Counter)

	p.readyRead.Store(true)

	p.pollCount++
	return p.pollCount
}

func (p *Poller) Get() (int, map[string]Gauge, map[string]Counter) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	gauge := make(map[string]Gauge)
	maps.Copy(gauge, p.metrics.Gauge)

	counter := make(map[string]Counter)
	maps.Copy(counter, p.metrics.Counter)

	return p.pollCount, gauge, counter
}

func (p *Poller) ReadyRead() bool {
	return p.readyRead.Load()
}
