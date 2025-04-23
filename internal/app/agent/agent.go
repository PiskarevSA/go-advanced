package agent

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	"github.com/PiskarevSA/go-advanced/internal/app/agent/workers"
)

type Agent struct{}

func NewAgent() *Agent {
	return &Agent{}
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

	// report metrics to server periodically
	reporterPool := workers.NewReporterPool(
		&wg, config.RateLimit, metricsChan, config.ServerAddress, config.Key)
	reporterPool.StartReporters(ctx)

	// Wait for all goroutines to finish
	wg.Wait()
}
