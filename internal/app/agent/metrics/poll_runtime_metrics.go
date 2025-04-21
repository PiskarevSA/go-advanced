package metrics

import (
	"crypto/rand"
	"math"
	"math/big"
	"runtime"
)

func PollRuntimeMetrics(gaugeMap map[string]Gauge, counter map[string]Counter) {
	// runtime metrics
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	gaugeMap["Alloc"] = Gauge(ms.Alloc)
	gaugeMap["BuckHashSys"] = Gauge(ms.BuckHashSys)
	gaugeMap["Frees"] = Gauge(ms.Frees)
	gaugeMap["GCCPUFraction"] = Gauge(ms.GCCPUFraction)
	gaugeMap["GCSys"] = Gauge(ms.GCSys)
	gaugeMap["HeapAlloc"] = Gauge(ms.HeapAlloc)
	gaugeMap["HeapIdle"] = Gauge(ms.HeapIdle)
	gaugeMap["HeapInuse"] = Gauge(ms.HeapInuse)
	gaugeMap["HeapObjects"] = Gauge(ms.HeapObjects)
	gaugeMap["HeapReleased"] = Gauge(ms.HeapReleased)
	gaugeMap["HeapSys"] = Gauge(ms.HeapSys)
	gaugeMap["LastGC"] = Gauge(ms.LastGC)
	gaugeMap["Lookups"] = Gauge(ms.Lookups)
	gaugeMap["MCacheInuse"] = Gauge(ms.MCacheInuse)
	gaugeMap["MCacheSys"] = Gauge(ms.MCacheSys)
	gaugeMap["MSpanInuse"] = Gauge(ms.MSpanInuse)
	gaugeMap["MSpanSys"] = Gauge(ms.MSpanSys)
	gaugeMap["Mallocs"] = Gauge(ms.Mallocs)
	gaugeMap["NextGC"] = Gauge(ms.NextGC)
	gaugeMap["NumForcedGC"] = Gauge(ms.NumForcedGC)
	gaugeMap["NumGC"] = Gauge(ms.NumGC)
	gaugeMap["OtherSys"] = Gauge(ms.OtherSys)
	gaugeMap["PauseTotalNs"] = Gauge(ms.PauseTotalNs)
	gaugeMap["StackInuse"] = Gauge(ms.StackInuse)
	gaugeMap["StackSys"] = Gauge(ms.StackSys)
	gaugeMap["Sys"] = Gauge(ms.Sys)
	gaugeMap["TotalAlloc"] = Gauge(ms.TotalAlloc)
	// custom metrics
	randomInt64, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	gaugeMap["RandomValue"] = Gauge(randomInt64.Int64())
	counter["PollCount"] += 1
}
