package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/models"
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
			Timeout: 15 * time.Second,
			Transport: &retryableTransport{
				transport: &http.Transport{},
			},
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
		slog.Info("[signal handler] Waiting for an interrupt signal...")

		// Wait for an interrupt signal to initiate graceful shutdown
		<-sigChan

		// Handle shutdown signal (Ctrl+C or SIGTERM)
		slog.Info("[signal handler] Received shutdown signal")

		// Cancel the context to notify all goroutines to stop
		cancel()
		slog.Info("[signal handler] Cancel func called")
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
	a.startReporter(ctx, &wg, reportInterval, config.ServerAddress, config.Key)

	// Wait for all goroutines to finish
	wg.Wait()
}

func (a *Agent) startPoller(ctx context.Context, wg *sync.WaitGroup, pollInterval time.Duration) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[poller] start ")

		for {
			pollCount := a.metrics.Poll()
			slog.Info("[poller] polled", "pollCount", pollCount)
			a.readyRead.Store(true)
			// sleep pollInterval or interrupt
			for t := time.Duration(0); t < pollInterval; t += updateInterval {
				select {
				case <-ctx.Done():
					// Handle context cancellation (graceful shutdown)
					slog.Info("[poller] stopping", "reason", ctx.Err())
					return
				default:
					time.Sleep(updateInterval)
				}
			}
		}
	}()
}

func (a *Agent) startReporter(ctx context.Context, wg *sync.WaitGroup,
	reportInterval time.Duration, serverAddress string, key string,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[reporter] start")
		// wait for first poll
		for !a.readyRead.Load() {
			slog.Info("[reporter] waiting for first poll")
			time.Sleep(time.Microsecond)
		}
		for {
			pollCount, gauge, counter := a.metrics.Get()
			// report
			if err := a.Report(serverAddress, gauge, counter, key); err != nil {
				slog.Error("[reporter] report failed", "pollCount", pollCount, "error", err)
			} else {
				slog.Info("[reporter] report succeeded", "pollCount", pollCount)
			}
			// sleep reportInterval or interrupt
			for t := time.Duration(0); t < reportInterval; t += updateInterval {
				select {
				case <-ctx.Done():
					// Handle context cancellation (graceful shutdown)
					slog.Info("[reporter] stopping", "reason", ctx.Err())
					return
				default:
					time.Sleep(updateInterval)
				}
			}
		}
	}()
}

func (a *Agent) Report(serverAddress string, gauge map[string]gauge,
	counter map[string]counter, key string,
) error {
	url := "http://" + serverAddress + "/updates/"

	metrics := make([]models.Metric, 0, len(gauge)+len(counter))
	for key, gauge := range gauge {
		value := float64(gauge)
		m := models.Metric{
			ID:    key,
			MType: "gauge",
			Value: &value,
		}
		metrics = append(metrics, m)
	}

	for key, counter := range counter {
		delta := int64(counter)
		m := models.Metric{
			ID:    key,
			MType: "counter",
			Delta: &delta,
		}
		metrics = append(metrics, m)
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	if err := a.ReportToURL(url, body, key); err != nil {
		return err
	}
	return nil
}

func (a *Agent) ReportToURL(url string, body []byte, key string) error {
	compressedBodyBuffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(compressedBodyBuffer)
	// write compressed body to buffer
	if _, err := gzipWriter.Write(body); err != nil {
		return fmt.Errorf("gzipWriter.Write(): %w", err)
	}
	// flush any unwritten data to buffer
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("gzipWriter.Close(): %w", err)
	}

	var hexSum string
	if len(key) > 0 {
		h := hmac.New(sha256.New, []byte(key))
		compressedBody := compressedBodyBuffer.Bytes()
		h.Write(compressedBody)
		sign := h.Sum(nil)
		hexSum = hex.EncodeToString(sign[:])
	}

	req, err := http.NewRequest(http.MethodPost, url, compressedBodyBuffer)
	if err != nil {
		return fmt.Errorf("http.NewRequest(): %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if len(hexSum) > 0 {
		req.Header.Set("HashSHA256", hexSum)
	}
	res, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("httpClient.Do(): %w", err)
	}
	defer res.Body.Close()

	// The default HTTP client's Transport may not
	// reuse HTTP/1.x "keep-alive" TCP connections if the Body is
	// not read to completion and closed.
	_, err = io.Copy(io.Discard, res.Body)
	if err != nil {
		return fmt.Errorf("io.Copy(): %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %v returns %v", url, res.Status)
	}

	return nil
}
