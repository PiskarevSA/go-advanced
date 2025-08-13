package workers

import (
	"context"
	"log/slog"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
)

type Reporter interface {
	Start(ctx context.Context)
}

type ReporterCreator interface {
	Do(wg *sync.WaitGroup, metricsChan <-chan metrics.Metrics, reporterIndex int) Reporter
}

type ReporterPool struct {
	wg              *sync.WaitGroup
	rateLimit       int
	metricsChan     <-chan metrics.Metrics
	reporterCreator ReporterCreator
}

func NewReporterPool(
	wg *sync.WaitGroup, rateLimit int, metricsChan <-chan metrics.Metrics,
	reporterCreator ReporterCreator,
) *ReporterPool {
	return &ReporterPool{
		wg:              wg,
		rateLimit:       rateLimit,
		metricsChan:     metricsChan,
		reporterCreator: reporterCreator,
	}
}

func (p *ReporterPool) StartReporters(ctx context.Context) error {
	if p.rateLimit < 1 {
		slog.Warn("[reporter pool] start flusher instead of reporters",
			"rateLimit", p.rateLimit)
		flusher := NewFlusher(p.wg, p.metricsChan)
		flusher.Start(ctx)
		return nil
	}

	for reporterIndex := range p.rateLimit {
		slog.Info("[reporter pool] start reporter",
			"reporterIndex", reporterIndex)
		reporter := p.reporterCreator.Do(p.wg, p.metricsChan, reporterIndex)
		reporter.Start(ctx)
	}
	return nil
}
