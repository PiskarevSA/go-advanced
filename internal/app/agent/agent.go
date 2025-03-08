package agent

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const updateInterval = 100 * time.Millisecond

type Agent struct {
	metrics   *metrics
	readyRead atomic.Bool
	stopped   atomic.Bool
}

func NewAgent() *Agent {
	return &Agent{
		metrics: newMetrics(),
	}
}

func (a *Agent) metricsReader(gauge map[string]gauge, counter map[string]counter) func(*metrics) {
	return func(m *metrics) {
		// runtime metrics
		gauge["Alloc"] = m.Alloc
		gauge["BuckHashSys"] = m.BuckHashSys
		gauge["Frees"] = m.Frees
		gauge["GCCPUFraction"] = m.GCCPUFraction
		gauge["GCSys"] = m.GCSys
		gauge["HeapAlloc"] = m.HeapAlloc
		gauge["HeapIdle"] = m.HeapIdle
		gauge["HeapInuse"] = m.HeapInuse
		gauge["HeapObjects"] = m.HeapObjects
		gauge["HeapReleased"] = m.HeapReleased
		gauge["HeapSys"] = m.HeapSys
		gauge["LastGC"] = m.LastGC
		gauge["Lookups"] = m.Lookups
		gauge["MCacheInuse"] = m.MCacheInuse
		gauge["MCacheSys"] = m.MCacheSys
		gauge["MSpanInuse"] = m.MSpanInuse
		gauge["MSpanSys"] = m.MSpanSys
		gauge["Mallocs"] = m.Mallocs
		gauge["NextGC"] = m.NextGC
		gauge["NumForcedGC"] = m.NumForcedGC
		gauge["NumGC"] = m.NumGC
		gauge["OtherSys"] = m.OtherSys
		gauge["PauseTotalNs"] = m.PauseTotalNs
		gauge["StackInuse"] = m.StackInuse
		gauge["StackSys"] = m.StackSys
		gauge["Sys"] = m.Sys
		gauge["TotalAlloc"] = m.TotalAlloc
		// custom metrics
		gauge["RandomValue"] = m.RandomValue
		counter["PollCount"] = m.PollCount
	}
}

// TODO PR #5
// В функции нарушен принцип единой ответственно. Стоит разнести на хелперы.
// Сейчас тут и конфиг, и poll и report.
//
// Также конфиг стоит парсить в мэйне
//
// В идеале должно стать как-то так. Названия от балды, придумай, как лучше
//
// func (a *Agent) Run() error {
// 	config, err := a.parseConfig()
// 	if err != nil {
// 		return err
// 	}
//
// 	ctx, cancel := a.setupSignalHandler()
// 	defer cancel()
//
// 	return a.startWorkers(ctx, config)
// }

// run agent successfully or return false immediately
func (a *Agent) Run(config *Config) bool {
	// set a.stopped on program interrupt requested
	// TODO PR #5
	// Сейчас горутины не завершаются, если поступил os.Interrupt.
	// Стоит воспользоваться контекстом с отменой и передавать этот контекст
	// внутрь методов, запускающих горутины
	//
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	//
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	//
	// go func() {
	// 	<-c
	// 	log.Println("[signal] Interrupt signal received")
	// 	a.stopped.Store(true)
	// 	cancel() // Завершаем все горутины
	// }()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		log.Println("[signal] waiting for interrupt signal from OS")
		defer wg.Done()
		for range c {
			a.stopped.Store(true)
			log.Println("[signal] Interrupt signal from OS received")
			break
		}
	}()

	// TODO PR #5
	// код из каждой горутины можно вынести в свой хелпер метод типа
	// a.pollMetrics и a.reportMetrics. Сделает код чище и приятнее.
	//
	// Типа такого
	//
	// Запускает воркеры для опроса и отправки метрик
	// func (a *Agent) startWorkers(ctx context.Context, config Config) error {
	// 	var wg sync.WaitGroup
	//
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		a.pollMetrics(ctx, time.Duration(config.PollIntervalSec)*time.Second)
	// 	}()
	//
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		a.reportMetrics(ctx, time.Duration(config.ReportIntervalSec)*time.Second, config.ServerAddress)
	// 	}()
	//
	// 	wg.Wait()
	// 	return nil
	// }
	//
	// TODO PR #5
	// Также в циклах for нам нужно будет добавить select, чтобы слушать
	// отмену контекста
	// Пример:
	//
	// case <-ctx.Done():
	// 		log.Println("[poller] shutdown")
	// 		return

	// poll metrics periodically
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[poller] start ")
		pollInterval := time.Duration(config.PollIntervalSec) * time.Second
		for {
			pollCount := a.metrics.Poll()
			log.Println("[poller] polled", pollCount)
			a.readyRead.Store(true)
			// sleep pollInterval or interrupt
			for t := updateInterval; t < pollInterval; t += updateInterval {
				if a.stopped.Load() {
					log.Println("[poller] shutdown")
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
		log.Println("[reporter] start")
		// wait for first poll
		for !a.readyRead.Load() {
			log.Println("[reporter] waiting for first poll")
			time.Sleep(time.Microsecond)
		}
		reportInterval := time.Duration(config.ReportIntervalSec) * time.Second
		for {
			gauge := make(map[string]gauge)
			counter := make(map[string]counter)
			a.metrics.Read(a.metricsReader(gauge, counter))
			// report
			a.Report(config.ServerAddress, gauge, counter)
			log.Println("[reporter] reported", a.metrics.PollCount)
			// sleep reportInterval or interrupt
			for t := updateInterval; t < reportInterval; t += updateInterval {
				if a.stopped.Load() {
					log.Println("[reporter] shutdown")
					return
				}
				time.Sleep(updateInterval)
			}
		}
	}()
	wg.Wait()
	return true
}

func (a *Agent) Report(
	serverAddress string, gauge map[string]gauge, counter map[string]counter,
) {
	urls := make([]string, 0, len(gauge)+len(counter))
	for key, gauge := range gauge {
		urls = append(urls, strings.Join(
			[]string{"http://" + serverAddress, "update", "gauge", key, fmt.Sprint(gauge)}, "/"))
	}
	for key, counter := range counter {
		urls = append(urls, strings.Join(
			[]string{"http://" + serverAddress, "update", "counter", key, fmt.Sprint(counter)}, "/"))
	}
	var (
		firstError error
		errorCount int
	)
	for _, url := range urls {
		res, err := a.ReportToURL(url)
		// TODO PR #5
		// лучше просто сначала проверять на ошибку, а после проверки
		// сделать defer res.Body.Close(), тогда не нужна проверка на nil
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
			log.Println("- interrupt reporting")
			return
		}
	}
	if errorCount > 0 {
		message := fmt.Sprintf("[reporter] %v", firstError)
		if errorCount > 1 {
			message += fmt.Sprintf(" (and %v more errors)", errorCount-1)
		}
		log.Println(message)
	}
}

// TODO PR #5
// Текущий код создает новый HTTP-клиент на каждый запрос, что неэффективно.
// Используем http.Client (желательно с таймаутом)
//
//	type Agent struct {
//		httpClient *http.Client
//	}
//
//	a := &Agent{
//		httpClient: &http.Client{
//			Timeout: 5 * time.Second,
//		},
//	}
func (a *Agent) ReportToURL(url string) (*http.Response, error) {
	res, err := http.Post(url, "text/plain", http.NoBody)
	if res != nil {
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("POST %v returns %v", url, res.Status)
		}
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return res, err
}
