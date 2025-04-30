package workers

import (
	"context"
	"log/slog"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
)

type Flusher struct {
	wg          *sync.WaitGroup
	metricsChan <-chan metrics.Metrics
}

func NewFlusher(wg *sync.WaitGroup, metricsChan <-chan metrics.Metrics) *Flusher {
	return &Flusher{
		wg:          wg,
		metricsChan: metricsChan,
	}
}

func (f *Flusher) Start(ctx context.Context) {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for {
			select {
			case <-ctx.Done():
				slog.Info("[flusher] stopping",
					"reason", ctx.Err())
				return
			case metric, ok := <-f.metricsChan:
				if !ok {
					slog.Info("[flusher] stopping",
						"reason", "metrics channel closed")
					return
				}
				slog.Info("[flusher] flush",
					"metric", metric)
			}
		}
	}()
}
