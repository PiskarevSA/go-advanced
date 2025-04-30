package workers

import (
	"context"
	"log/slog"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
)

type ReporterPool struct {
	wg            *sync.WaitGroup
	rateLimit     int
	metricsChan   <-chan metrics.Metrics
	serverAddress string
	key           string
}

func NewReporterPool(
	wg *sync.WaitGroup, rateLimit int, metricsChan <-chan metrics.Metrics,
	serverAddress string, key string,
) *ReporterPool {
	return &ReporterPool{
		wg:            wg,
		rateLimit:     rateLimit,
		metricsChan:   metricsChan,
		serverAddress: serverAddress,
		key:           key,
	}
}

func (p *ReporterPool) StartReporters(ctx context.Context) {
	if p.rateLimit < 1 {
		slog.Warn("[reporter pool] start flusher instead of reporters",
			"rateLimit", p.rateLimit)
		flusher := NewFlusher(p.wg, p.metricsChan)
		flusher.Start(ctx)
		return
	}

	for reporterIndex := range p.rateLimit {
		slog.Info("[reporter pool] start reporter",
			"reporterIndex", reporterIndex)
		reporter := NewReporter(p.wg, reporterIndex,
			p.metricsChan, p.serverAddress, p.key)
		reporter.Start(ctx)
	}
}
