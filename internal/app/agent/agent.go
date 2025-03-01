package agent

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	updateInterval = 100 * time.Millisecond
	serverAddress  = "http://localhost:8080"
)

type Agent struct {
	metrics   *metrics
	gauge     map[string]gauge
	counter   map[string]counter
	readyRead atomic.Bool
	stopped   atomic.Bool
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
	// set a.stopped on program interrupt requested
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		fmt.Println("[signal] waiting for interrupt signal from OS")
		defer wg.Done()
		for range c {
			a.stopped.Store(true)
			fmt.Println("[signal] Interrupt signal from OS received")
			break
		}
	}()

	// poll metrics periodically
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("[poller] start ")
		for {
			a.metrics.Poll()
			fmt.Println("[poller] polled ")
			a.readyRead.Store(true)
			// sleep pollInterval or interrupt
			for t := updateInterval; t < pollInterval; t += updateInterval {
				if a.stopped.Load() {
					fmt.Println("[poller] shutdown")
					return
				}
				time.Sleep(updateInterval)
			}
		}
	}()

	// report metrics to server periodically
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("[reporter] start")
		// wait for first poll
		for !a.readyRead.Load() {
			fmt.Println("[reporter] waiting for first poll")
			time.Sleep(time.Microsecond)
		}
		for {
			a.metrics.Read(a.metricsReader())
			// report
			a.Report()
			fmt.Println("[reporter] reported")
			// sleep reportInterval or interrupt
			for t := updateInterval; t < reportInterval; t += updateInterval {
				if a.stopped.Load() {
					fmt.Println("[reporter] shutdown")
					return
				}
				time.Sleep(updateInterval)
			}
		}
	}()
	wg.Wait()
	return nil
}

func (a *Agent) Report() {
	urls := make([]string, 0, len(a.gauge)+len(a.counter))
	for key, gauge := range a.gauge {
		urls = append(urls, strings.Join(
			[]string{serverAddress, "gauge", key, fmt.Sprint(gauge)}, "/"))
	}
	for key, counter := range a.counter {
		urls = append(urls, strings.Join(
			[]string{serverAddress, "counter", key, fmt.Sprint(counter)}, "/"))
	}
	var (
		firstError error
		errorCount int
	)
	for _, url := range urls {
		res, err := a.ReportToURL(url)
		if res != nil {
			res.Body.Close()
		}
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			errorCount += 1
		}
		if a.stopped.Load() {
			fmt.Println("- interrupt reporting")
			return
		}
	}
	if errorCount > 0 {
		message := fmt.Sprintf("[reporter] %v", firstError)
		if errorCount > 1 {
			message += fmt.Sprintf(" (and %v more errors)", errorCount-1)
		}
		fmt.Println(message)
	}
}

func (a *Agent) ReportToURL(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return res, err
}
