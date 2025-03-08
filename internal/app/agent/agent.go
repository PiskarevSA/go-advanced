package agent

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caarlos0/env/v6"
)

const updateInterval = 100 * time.Millisecond

// TODO PR #5
// для конфига можно сделать отдельный пакет на уровне с internal
//
// config/config.go
// internal/...
type Config struct {
	PollIntervalSec   int    `env:"POLL_INTERVAL"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL"`
	ServerAddress     string `env:"ADDRESS"`
}

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

// TODO PR #5
// В коде metricsReader и Report используется изменение мап без блокировки, что
// в многопоточной среде приведет к панике (fatal error: concurrent map writes).
// Нужно использовать мьютексы, если хотим делать это всё дело асихнронно
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

// run agent successfully or return error to panic in the main()
func (a *Agent) Run() error {
	var config Config
	// flags takes less priority according to task description
	flag.IntVar(&config.PollIntervalSec, "p", 2,
		"interval between polling metrics, seconds; env: POLL_INTERVAL")
	flag.IntVar(&config.ReportIntervalSec, "r", 10,
		"interval between sending metrics to server, seconds; env: REPORT_INTERVAL")
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080",
		"server address; env: ADDRESS")
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		return nil
	}
	log.Printf("config after flags: %+v\n", config)

	// enviromnent takes higher priority according to task description
	err := env.Parse(&config)
	if err != nil {
		log.Println(err)
		flag.Usage()
		return nil
	}
	log.Printf("config after env: %+v\n", config)

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
			a.metrics.Poll()
			log.Println("[poller] polled", a.metrics.PollCount)
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
			a.metrics.Read(a.metricsReader())
			// report
			a.Report(config.ServerAddress)
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
	return nil
}

func (a *Agent) Report(serverAddress string) {
	urls := make([]string, 0, len(a.gauge)+len(a.counter))
	for key, gauge := range a.gauge {
		urls = append(urls, strings.Join(
			[]string{"http://" + serverAddress, "update", "gauge", key, fmt.Sprint(gauge)}, "/"))
	}
	for key, counter := range a.counter {
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
