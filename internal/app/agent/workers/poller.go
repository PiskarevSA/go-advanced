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

func (p *PollerLauncher) startPoll(
	ctx context.Context, poller *metrics.Poller, name string,
) {
	p.wg.Add(1)

	go func() {
		withPrefix := func(msg string) string {
			return "[" + name + "] " + msg
		}

		defer p.wg.Done()
		slog.Info(withPrefix("start"))

		// make first poll instantly and use ticker after that
		firstPoll := make(chan struct{})
		close(firstPoll)
		ticker := time.NewTicker(p.interval)

		poll := func() {
			pollCount := poller.Poll()
			slog.Info(withPrefix("polled"), "pollCount", pollCount)
		}

		stop := func() {
			slog.Info(withPrefix("stopping"), "reason", ctx.Err())
			ticker.Stop()
		}

		for {
			select {
			case <-ctx.Done():
				stop()
				return
			case <-firstPoll:
				poll()
				firstPoll = nil
			case <-ticker.C:
				poll()
			}
		}
	}()
}

func (p *PollerLauncher) StartPollRuntime(ctx context.Context) *metrics.Poller {
	poller := metrics.NewPoller(metrics.PollRuntimeMetrics)
	p.startPoll(ctx, poller, "runtime poller")
	return poller
}

func (p *PollerLauncher) StartPollGopsutil(ctx context.Context) *metrics.Poller {
	poller := metrics.NewPoller(metrics.PollGopsutilMetrics)
	p.startPoll(ctx, poller, "gopsutil poller")
	return poller
}
