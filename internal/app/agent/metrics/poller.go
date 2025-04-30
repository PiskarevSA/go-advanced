package metrics

import (
	"maps"
	"sync"
)

type PollFunc func(gauge map[string]Gauge, counter map[string]Counter)

type Poller struct {
	mutex           sync.RWMutex
	pollFunc        PollFunc
	metrics         Metrics
	readyRead       chan struct{}
	readyReadCloser func()
	pollCount       int
}

func NewPoller(pollFunc PollFunc) *Poller {
	readyRead := make(chan struct{})
	return &Poller{
		pollFunc:  pollFunc,
		metrics:   *NewMetrics(),
		readyRead: readyRead,
		readyReadCloser: sync.OnceFunc(func() {
			close(readyRead)
		}),
	}
}

// returns poll count
func (p *Poller) Poll() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.pollFunc(p.metrics.Gauge, p.metrics.Counter)

	p.readyReadCloser()

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

func (p *Poller) ReadyRead() chan struct{} {
	return p.readyRead
}
