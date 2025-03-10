package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const updateInterval = 100 * time.Millisecond

type Agent struct {
	httpClient *http.Client
	metrics    *metrics
	readyRead  atomic.Bool
}

func NewAgent() *Agent {
	return &Agent{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		metrics: newMetrics(),
	}
}

// run agent successfully or return false immediately
func (a *Agent) Run(config *Config) bool {
	ctx, cancel := a.setupSignalHandler()
	defer cancel() // Ensure cancel is called at the end to clean up

	a.startWorkers(ctx, config)

	// agent will never fails actually
	return true
}

func (a *Agent) setupSignalHandler() (context.Context, context.CancelFunc) {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Channel to listen for system signals (e.g., Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Wait for an interrupt signal to initiate graceful shutdown
		<-sigChan

		// Handle shutdown signal (Ctrl+C or SIGTERM)
		log.Println("Received shutdown signal. Shutting down gracefully...")

		// Cancel the context to notify all goroutines to stop
		cancel()
	}()
	return ctx, cancel
}

func (a *Agent) startWorkers(ctx context.Context, config *Config) {
	// Wait group to ensure all goroutines finish before exiting
	var wg sync.WaitGroup

	// poll metrics periodically
	pollInterval := time.Duration(config.PollIntervalSec) * time.Second
	a.startPoller(ctx, &wg, pollInterval)

	// report metrics to server periodically
	reportInterval := time.Duration(config.ReportIntervalSec) * time.Second
	a.startReporter(ctx, &wg, reportInterval, config.ServerAddress)

	// Wait for all goroutines to finish
	wg.Wait()
}

func (a *Agent) startPoller(ctx context.Context, wg *sync.WaitGroup, pollInterval time.Duration) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[poller] start ")

		for {
			pollCount := a.metrics.Poll()
			log.Println("[poller] polled", pollCount)
			a.readyRead.Store(true)
			// sleep pollInterval or interrupt
			for t := time.Duration(0); t < pollInterval; t += updateInterval {
				select {
				case <-ctx.Done():
					// Handle context cancellation (graceful shutdown)
					log.Printf("[poller] stopping: %v\n", ctx.Err())
					return
				default:
					time.Sleep(updateInterval)
				}
			}
		}
	}()
}

func (a *Agent) startReporter(
	ctx context.Context, wg *sync.WaitGroup, reportInterval time.Duration, serverAddress string,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[reporter] start")
		// wait for first poll
		for !a.readyRead.Load() {
			log.Println("[reporter] waiting for first poll")
			time.Sleep(time.Microsecond)
		}
		for {
			pollCount, gauge, counter := a.metrics.Get()
			// report
			a.Report(ctx, serverAddress, gauge, counter)
			log.Println("[reporter] reported", pollCount)
			// sleep reportInterval or interrupt
			for t := time.Duration(0); t < reportInterval; t += updateInterval {
				select {
				case <-ctx.Done():
					// Handle context cancellation (graceful shutdown)
					log.Printf("[reporter] stopping: %v\n", ctx.Err())
					return
				default:
					time.Sleep(updateInterval)
				}
			}
		}
	}()
}

// sending a report consists of multiple HTTP requests; before each of them, it
// is checked whether it is necessary to abort the execution
func (a *Agent) Report(
	ctx context.Context, serverAddress string, gauge map[string]gauge, counter map[string]counter,
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
	log.Println("[reporter] start reporting")
	for _, url := range urls {
		select {
		case <-ctx.Done():
			// Handle context cancellation (graceful shutdown)
			log.Printf("[reporter] interrupt reporting: %v\n", ctx.Err())
			return
		default:
			// send next metric
			err := a.ReportToURL(url)
			if err != nil {
				if errorCount == 0 {
					firstError = err
				}
				errorCount += 1
			}
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

func (a *Agent) ReportToURL(url string) error {
	res, err := a.httpClient.Post(url, "text/plain", http.NoBody)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// The default HTTP client's Transport may not
	// reuse HTTP/1.x "keep-alive" TCP connections if the Body is
	// not read to completion and closed.
	_, err = io.Copy(io.Discard, res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %v returns %v", url, res.Status)
	}

	return nil
}
