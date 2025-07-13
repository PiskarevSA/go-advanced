package workers

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	"github.com/PiskarevSA/go-advanced/internal/middleware"
)

type ReporterPool struct {
	wg            *sync.WaitGroup
	rateLimit     int
	metricsChan   <-chan metrics.Metrics
	serverAddress string
	key           string
	cryptoKey     string
}

func NewReporterPool(
	wg *sync.WaitGroup, rateLimit int, metricsChan <-chan metrics.Metrics,
	serverAddress string, key string, cryptoKey string,
) *ReporterPool {
	return &ReporterPool{
		wg:            wg,
		rateLimit:     rateLimit,
		metricsChan:   metricsChan,
		serverAddress: serverAddress,
		key:           key,
		cryptoKey:     cryptoKey,
	}
}

func (p *ReporterPool) StartReporters(ctx context.Context) bool {
	if p.rateLimit < 1 {
		slog.Warn("[reporter pool] start flusher instead of reporters",
			"rateLimit", p.rateLimit)
		flusher := NewFlusher(p.wg, p.metricsChan)
		flusher.Start(ctx)
		return true
	}

	var encoder func(*http.Request) error
	if len(p.cryptoKey) > 0 {
		var err error
		encoder, err = middleware.RSAEncoder(p.cryptoKey)
		if err != nil {
			slog.Error("[reporter pool] rsaencoder", "error", err.Error())
			return false
		}
	}

	for reporterIndex := range p.rateLimit {
		slog.Info("[reporter pool] start reporter",
			"reporterIndex", reporterIndex)
		reporter := NewReporter(p.wg, reporterIndex,
			p.metricsChan, p.serverAddress, p.key, encoder)
		reporter.Start(ctx)
	}
	return true
}
