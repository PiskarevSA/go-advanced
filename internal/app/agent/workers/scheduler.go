package workers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
)

type SchedulerLauncher struct {
	interval time.Duration
	wg       *sync.WaitGroup
}

func NewSchedulerLauncher(interval time.Duration, wg *sync.WaitGroup,
) *SchedulerLauncher {
	return &SchedulerLauncher{
		interval: interval,
		wg:       wg,
	}
}

func (l *SchedulerLauncher) StartScheduler(
	ctx context.Context, pollersMetrics []*metrics.Poller,
) <-chan metrics.Metrics {
	result := make(chan metrics.Metrics)

	l.wg.Add(1)

	go func() {
		defer l.wg.Done()
		slog.Info("[scheduler] start")
		// wait for first poll
		for i, pollerMetrics := range pollersMetrics {
			for !pollerMetrics.ReadyRead() {
				slog.Info("[scheduler] waiting for first poll", "metric index", i)
				time.Sleep(time.Microsecond)
			}
		}

		schedule := func(pollerIndex int, poller *metrics.Poller) {
			pollCount, gauge, counter := poller.Get()
			slog.Info("[scheduler] schedule", "pollerIndex", pollerIndex, "pollCount", pollCount)
			result <- metrics.Metrics{
				Gauge:   gauge,
				Counter: counter,
			}
			slog.Info("[scheduler] complete", "pollerIndex", pollerIndex, "pollCount", pollCount)
		}

		ticker := time.NewTicker(l.interval)
		stop := func() {
			slog.Info("[scheduler] stopping", "reason", ctx.Err())
			ticker.Stop()
			close(result)
		}

		// make first poll instantly
		for pollerIndex, pollerMetrics := range pollersMetrics {
			select {
			case <-ctx.Done():
				stop()
				return
			default:
				schedule(pollerIndex, pollerMetrics)
			}
		}

		// use ticker after that
		for pollerIndex, pollerMetrics := range pollersMetrics {
			for {
				select {
				case <-ctx.Done():
					stop()
					return
				case <-ticker.C:
					schedule(pollerIndex, pollerMetrics)
				}
			}
		}
	}()

	return result
}
