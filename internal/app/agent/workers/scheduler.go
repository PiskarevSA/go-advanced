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
		defer func() {
			slog.Info("[scheduler] stopping", "reason", ctx.Err())
			close(result)
			l.wg.Done()
		}()

		slog.Info("[scheduler] start")
		// wait for first poll
		for pollerIndex, pollerMetrics := range pollersMetrics {
			select {
			case <-ctx.Done():
				slog.Info("[scheduler] wait for first poll canceled",
					"pollerIndex", pollerIndex,
					"error", ctx.Err())
				return
			case <-pollerMetrics.ReadyRead():
			}
		}

		schedulePollerWithContext := func(ctx context.Context, pollerIndex int, poller *metrics.Poller) error {
			pollCount, gauge, counter := poller.Get()
			slog.Info("[scheduler] schedule",
				"pollerIndex", pollerIndex,
				"pollCount", pollCount)
			select {
			case <-ctx.Done():
				slog.Info("[scheduler] canceled",
					"pollerIndex", pollerIndex,
					"pollCount", pollCount,
					"error", ctx.Err())
				return ctx.Err()
			case result <- metrics.Metrics{
				Gauge:   gauge,
				Counter: counter,
			}:
				slog.Info("[scheduler] complete",
					"pollerIndex", pollerIndex,
					"pollCount", pollCount)
			}
			return nil
		}

		scheduleAllPollersWithContext := func(ctx context.Context) error {
			for pollerIndex, pollerMetrics := range pollersMetrics {
				err := schedulePollerWithContext(ctx, pollerIndex, pollerMetrics)
				if err != nil {
					return err
				}
			}
			return nil
		}

		// make first poll instantly
		if scheduleAllPollersWithContext(ctx) != nil {
			return
		}

		// use ticker after that
		ticker := time.NewTicker(l.interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if scheduleAllPollersWithContext(ctx) != nil {
					return
				}
			}
		}
	}()

	return result
}
