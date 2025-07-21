package workers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	rsamiddleware "github.com/PiskarevSA/go-advanced/internal/middleware/rsa"
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

func (p *ReporterPool) StartReporters(ctx context.Context) error {
	if p.rateLimit < 1 {
		slog.Warn("[reporter pool] start flusher instead of reporters",
			"rateLimit", p.rateLimit)
		flusher := NewFlusher(p.wg, p.metricsChan)
		flusher.Start(ctx)
		return nil
	}

	var encoder func(*http.Request) error
	if len(p.cryptoKey) > 0 {
		var err error
		encoder, err = rsamiddleware.Encoder(p.cryptoKey)
		if err != nil {
			return fmt.Errorf("rsaencoder: %w", err)
		}
	}

	for reporterIndex := range p.rateLimit {
		slog.Info("[reporter pool] start reporter",
			"reporterIndex", reporterIndex)
		reporter := NewReporter(p.wg, reporterIndex,
			p.metricsChan, p.serverAddress, p.key, encoder)
		reporter.Start(ctx)
	}
	return nil
}
