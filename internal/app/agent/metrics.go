package agent

import (
	"crypto/rand"
	"math"
	"math/big"
	"runtime"
	"sync"
)

type (
	gauge   float64
	counter int64
)

type metrics struct {
	mutex   sync.Mutex
	gauge   map[string]gauge
	counter map[string]counter
}

func newMetrics() *metrics {
	return &metrics{
		gauge:   make(map[string]gauge),
		counter: make(map[string]counter),
	}
}

// returns poll count
func (m *metrics) Poll() counter {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// runtime metrics
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	m.gauge["Alloc"] = gauge(ms.Alloc)
	m.gauge["BuckHashSys"] = gauge(ms.BuckHashSys)
	m.gauge["Frees"] = gauge(ms.Frees)
	m.gauge["GCCPUFraction"] = gauge(ms.GCCPUFraction)
	m.gauge["GCSys"] = gauge(ms.GCSys)
	m.gauge["HeapAlloc"] = gauge(ms.HeapAlloc)
	m.gauge["HeapIdle"] = gauge(ms.HeapIdle)
	m.gauge["HeapInuse"] = gauge(ms.HeapInuse)
	m.gauge["HeapObjects"] = gauge(ms.HeapObjects)
	m.gauge["HeapReleased"] = gauge(ms.HeapReleased)
	m.gauge["HeapSys"] = gauge(ms.HeapSys)
	m.gauge["LastGC"] = gauge(ms.LastGC)
	m.gauge["Lookups"] = gauge(ms.Lookups)
	m.gauge["MCacheInuse"] = gauge(ms.MCacheInuse)
	m.gauge["MCacheSys"] = gauge(ms.MCacheSys)
	m.gauge["MSpanInuse"] = gauge(ms.MSpanInuse)
	m.gauge["MSpanSys"] = gauge(ms.MSpanSys)
	m.gauge["Mallocs"] = gauge(ms.Mallocs)
	m.gauge["NextGC"] = gauge(ms.NextGC)
	m.gauge["NumForcedGC"] = gauge(ms.NumForcedGC)
	m.gauge["NumGC"] = gauge(ms.NumGC)
	m.gauge["OtherSys"] = gauge(ms.OtherSys)
	m.gauge["PauseTotalNs"] = gauge(ms.PauseTotalNs)
	m.gauge["StackInuse"] = gauge(ms.StackInuse)
	m.gauge["StackSys"] = gauge(ms.StackSys)
	m.gauge["Sys"] = gauge(ms.Sys)
	m.gauge["TotalAlloc"] = gauge(ms.TotalAlloc)
	// custom metrics
	randomInt64, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	m.gauge["RandomValue"] = gauge(randomInt64.Int64())
	m.counter["PollCount"] += 1

	return m.counter["PollCount"]
}

func (m *metrics) Get() (counter, map[string]gauge, map[string]counter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	gauge := make(map[string]gauge)
	for k, v := range m.gauge {
		gauge[k] = v
	}

	counter := make(map[string]counter)
	for k, v := range m.counter {
		counter[k] = v
	}

	return m.counter["PollCount"], gauge, counter
}
