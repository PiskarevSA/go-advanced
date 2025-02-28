package agent

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	pollInterval = 2 * time.Second
	sendInterval = 10 * time.Second
)

type (
	gauge   float64
	counter int64
)

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

func (m *metrics) Poll() {
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
}

func (m *metrics) Read(reader func(*metrics)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	reader(m)
}

type Agent struct {
	metrics   *metrics
	gauge     map[string]gauge
	counter   map[string]counter
	readyRead atomic.Bool
}

func NewAgent() *Agent {
	return &Agent{
		metrics: newMetrics(),
		gauge:   make(map[string]gauge),
		counter: make(map[string]counter),
	}
}

func (a *Agent) metricsReader() func(*metrics) {
	return func(m *metrics) {
		// runtime metrics
		a.gauge["Alloc"] = m.Alloc
		a.gauge["BuckHashSys"] = m.BuckHashSys
		a.gauge["Frees"] = m.Frees
		a.gauge["GCCPUFraction"] = m.GCCPUFraction
		a.gauge["GCSys"] = m.GCSys
		a.gauge["HeapAlloc"] = m.HeapAlloc
		a.gauge["HeapIdle"] = m.HeapIdle
		a.gauge["HeapInuse"] = m.HeapInuse
		a.gauge["HeapObjects"] = m.HeapObjects
		a.gauge["HeapReleased"] = m.HeapReleased
		a.gauge["HeapSys"] = m.HeapSys
		a.gauge["LastGC"] = m.LastGC
		a.gauge["Lookups"] = m.Lookups
		a.gauge["MCacheInuse"] = m.MCacheInuse
		a.gauge["MCacheSys"] = m.MCacheSys
		a.gauge["MSpanInuse"] = m.MSpanInuse
		a.gauge["MSpanSys"] = m.MSpanSys
		a.gauge["Mallocs"] = m.Mallocs
		a.gauge["NextGC"] = m.NextGC
		a.gauge["NumForcedGC"] = m.NumForcedGC
		a.gauge["NumGC"] = m.NumGC
		a.gauge["OtherSys"] = m.OtherSys
		a.gauge["PauseTotalNs"] = m.PauseTotalNs
		a.gauge["StackInuse"] = m.StackInuse
		a.gauge["StackSys"] = m.StackSys
		a.gauge["Sys"] = m.Sys
		a.gauge["TotalAlloc"] = m.TotalAlloc
		// custom metrics
		a.gauge["RandomValue"] = m.RandomValue
		a.counter["PollCount"] = m.PollCount
	}
}

// run agent successfully or return error to panic in the main()
func (a *Agent) Run() error {
	wg := sync.WaitGroup{}
	// poll metrics periodically
	wg.Add(1)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		for {
			if IsClosed(stop) {
				break
			}
			a.metrics.Poll()
			a.readyRead.Store(true)
			time.Sleep(pollInterval)
		}
		wg.Done()
	}()

	// send metrics to server periodically
	wg.Add(1)
	go func() {
		// wait for first poll
		for !a.readyRead.Load() {
			fmt.Println("waiting for first poll")
			time.Sleep(time.Microsecond)
		}
		for {
			if IsClosed(stop) {
				break
			}
			a.metrics.Read(a.metricsReader())
			// TODO send metrics to server
			fmt.Println("gauge:", a.gauge)
			fmt.Println("counter:", a.counter)
			time.Sleep(sendInterval)
		}
		wg.Done()
	}()
	wg.Wait()
	return nil
}

func IsClosed[T any](ch <-chan T) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}
