package workers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
)

type PollerLauncher struct {
	interval time.Duration
	wg       *sync.WaitGroup
}

func NewPollerLauncher(interval time.Duration, wg *sync.WaitGroup) *PollerLauncher {
	return &PollerLauncher{
		interval: interval,
		wg:       wg,
	}
}

func (l *PollerLauncher) startPoll(
	ctx context.Context, poller *metrics.Poller, name string,
) {
	l.wg.Add(1)

	go func() {
		withPrefix := func(msg string) string {
			return "[" + name + "] " + msg
		}

		defer func() {
			slog.Info(withPrefix("stopping"), "reason", ctx.Err())
			l.wg.Done()
		}()

		slog.Info(withPrefix("start"))

		poll := func() {
			pollCount := poller.Poll()
			slog.Info(withPrefix("polled"), "pollCount", pollCount)
		}

		// make first poll instantly
		select {
		case <-ctx.Done():
			return
		default:
			poll()
		}

		// use ticker after that
		ticker := time.NewTicker(l.interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll()
			}
		}
	}()
}

func (l *PollerLauncher) StartPollRuntime(ctx context.Context) *metrics.Poller {
	poller := metrics.NewPoller(metrics.PollRuntimeMetrics)
	l.startPoll(ctx, poller, "runtime poller")
	return poller
}

func (l *PollerLauncher) StartPollGopsutil(ctx context.Context) *metrics.Poller {
	poller := metrics.NewPoller(metrics.PollGopsutilMetrics)
	l.startPoll(ctx, poller, "gopsutil poller")
	return poller
}
