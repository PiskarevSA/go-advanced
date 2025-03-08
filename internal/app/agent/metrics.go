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

// TODO PR #5
// структура как-то излишне выглядит. Не лучше сделать мапу, где ключ
// просто название метрики? Может и не лучше, но ты подумай
type metrics struct {
	mutex sync.Mutex
	// runtime metrics
	Alloc         gauge
	BuckHashSys   gauge
	Frees         gauge
	GCCPUFraction gauge
	GCSys         gauge
	HeapAlloc     gauge
	HeapIdle      gauge
	HeapInuse     gauge
	HeapObjects   gauge
	HeapReleased  gauge
	HeapSys       gauge
	LastGC        gauge
	Lookups       gauge
	MCacheInuse   gauge
	MCacheSys     gauge
	MSpanInuse    gauge
	MSpanSys      gauge
	Mallocs       gauge
	NextGC        gauge
	NumForcedGC   gauge
	NumGC         gauge
	OtherSys      gauge
	PauseTotalNs  gauge
	StackInuse    gauge
	StackSys      gauge
	Sys           gauge
	TotalAlloc    gauge
	// custom metrics
	RandomValue gauge
	PollCount   counter
}

func newMetrics() *metrics {
	return &metrics{}
}

// returns poll count
func (m *metrics) Poll() counter {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// runtime metrics
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	m.Alloc = gauge(ms.Alloc)
	m.BuckHashSys = gauge(ms.BuckHashSys)
	m.Frees = gauge(ms.Frees)
	m.GCCPUFraction = gauge(ms.GCCPUFraction)
	m.GCSys = gauge(ms.GCSys)
	m.HeapAlloc = gauge(ms.HeapAlloc)
	m.HeapIdle = gauge(ms.HeapIdle)
	m.HeapInuse = gauge(ms.HeapInuse)
	m.HeapObjects = gauge(ms.HeapObjects)
	m.HeapReleased = gauge(ms.HeapReleased)
	m.HeapSys = gauge(ms.HeapSys)
	m.LastGC = gauge(ms.LastGC)
	m.Lookups = gauge(ms.Lookups)
	m.MCacheInuse = gauge(ms.MCacheInuse)
	m.MCacheSys = gauge(ms.MCacheSys)
	m.MSpanInuse = gauge(ms.MSpanInuse)
	m.MSpanSys = gauge(ms.MSpanSys)
	m.Mallocs = gauge(ms.Mallocs)
	m.NextGC = gauge(ms.NextGC)
	m.NumForcedGC = gauge(ms.NumForcedGC)
	m.NumGC = gauge(ms.NumGC)
	m.OtherSys = gauge(ms.OtherSys)
	m.PauseTotalNs = gauge(ms.PauseTotalNs)
	m.StackInuse = gauge(ms.StackInuse)
	m.StackSys = gauge(ms.StackSys)
	m.Sys = gauge(ms.Sys)
	m.TotalAlloc = gauge(ms.TotalAlloc)
	// custom metrics
	randomInt64, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	m.RandomValue = gauge(randomInt64.Int64())
	m.PollCount += 1
	return m.PollCount
}

func (m *metrics) Read(reader func(*metrics)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	reader(m)
}
