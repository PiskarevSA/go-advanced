package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	"github.com/PiskarevSA/go-advanced/internal/app/agent/workers"
	rsamiddleware "github.com/PiskarevSA/go-advanced/internal/middleware/rsa"
	"github.com/PiskarevSA/go-advanced/internal/middleware/subnet"
)

type Agent struct{}

func NewAgent() *Agent {
	return &Agent{}
}

// run agent successfully or return false immediately
func (a *Agent) Run(config *Config) bool {
	ctx, cancel := a.setupSignalHandler()
	defer cancel() // Ensure cancel is called at the end to clean up

	if err := a.startWorkers(ctx, config); err != nil {
		slog.Error("[main] failed to start workers", "error", err.Error())
		return false
	}
	return true
}

func (a *Agent) setupSignalHandler() (context.Context, context.CancelFunc) {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Channel to listen for system signals (e.g., Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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

func (a *Agent) startWorkers(ctx context.Context, config *Config) error {
	// Wait group to ensure all goroutines finish before exiting
	var wg sync.WaitGroup

	// poll metrics periodically
	pollInterval := time.Duration(config.PollIntervalSec) * time.Second
	pollerLauncher := workers.NewPollerLauncher(pollInterval, &wg)
	runtimePollerMetrics := pollerLauncher.StartPollRuntime(ctx)
	gopsutilPollerMetrics := pollerLauncher.StartPollGopsutil(ctx)

	// schedule metrics for reporting periodically
	reportInterval := time.Duration(config.ReportIntervalSec) * time.Second
	schedulerLauncher := workers.NewSchedulerLauncher(
		reportInterval, &wg)
	metricsChan := schedulerLauncher.StartScheduler(ctx, []*metrics.Poller{
		runtimePollerMetrics,
		gopsutilPollerMetrics,
	})

	// hide reporter mode (rest or grpc) from reporter pool
	reporterCreator, err := a.makeReporterCreator(config)
	if err != nil {
		return fmt.Errorf("make reporter creator: %w", err)
	}

	// report metrics to server periodically
	reporterPool := workers.NewReporterPool(
		&wg, config.RateLimit, metricsChan, reporterCreator)
	if err := reporterPool.StartReporters(ctx); err != nil {
		return fmt.Errorf("start reporters: %w", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	return nil
}

func (a *Agent) makeReporterCreator(config *Config) (workers.ReporterCreator, error) {
	if config.UseGrpcMode {
		creator := RestReporterCreator{
			config: config,
		}
		return &creator, nil

	} else {
		setRealIP, err := subnet.SetHeader()
		if err != nil {
			return nil, fmt.Errorf("subnet middleware: %w", err)
		}

		var encoder func(*http.Request) error
		if len(config.CryptoKey) > 0 {
			var err error
			encoder, err = rsamiddleware.Encoder(config.CryptoKey)
			if err != nil {
				return nil, fmt.Errorf("rsaencoder: %w", err)
			}
		}

		creator := RestReporterCreator{
			config:    config,
			setRealIP: setRealIP,
			encoder:   encoder,
		}
		return &creator, nil
	}
}

type RestReporterCreator struct {
	config    *Config
	setRealIP func(*http.Request)
	encoder   func(*http.Request) error
}

func (c *RestReporterCreator) Do(
	wg *sync.WaitGroup, metricsChan <-chan metrics.Metrics, reporterIndex int,
) workers.Reporter {
	return workers.NewReporter(wg, reporterIndex,
		metricsChan, c.config.ServerAddress,
		c.config.Key, c.setRealIP, c.encoder)
}

type GrpcReporterCreator struct {
	config *Config
}

func (c *GrpcReporterCreator) Do(
	wg *sync.WaitGroup, metricsChan <-chan metrics.Metrics, reporterIndex int,
) workers.Reporter {
	return workers.NewGrpcReporter(wg, reporterIndex,
		metricsChan, c.config.GrpcServerAddress)
}
