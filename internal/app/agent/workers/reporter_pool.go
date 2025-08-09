package workers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	rsamiddleware "github.com/PiskarevSA/go-advanced/internal/middleware/rsa"
	"github.com/PiskarevSA/go-advanced/internal/middleware/subnet"
)

type ReporterPool struct {
	wg                *sync.WaitGroup
	rateLimit         int
	metricsChan       <-chan metrics.Metrics
	serverAddress     string
	grpcServerAddress string
	workMode          string
	key               string
	cryptoKey         string
}

func NewReporterPool(
	wg *sync.WaitGroup, rateLimit int, metricsChan <-chan metrics.Metrics,
	serverAddress string, grpcServerAddress string, workMode string,
	key string, cryptoKey string,
) *ReporterPool {
	return &ReporterPool{
		wg:                wg,
		rateLimit:         rateLimit,
		metricsChan:       metricsChan,
		serverAddress:     serverAddress,
		grpcServerAddress: grpcServerAddress,
		workMode:          workMode,
		key:               key,
		cryptoKey:         cryptoKey,
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

	setRealIP, err := subnet.SetHeader()
	if err != nil {
		return fmt.Errorf("subnet middleware: %w", err)
	}

	var encoder func(*http.Request) error
	if len(p.cryptoKey) > 0 {
		var err error
		encoder, err = rsamiddleware.Encoder(p.cryptoKey)
		if err != nil {
			return fmt.Errorf("rsaencoder: %w", err)
		}
	}

	var reporter interface {
		Start(ctx context.Context)
	}

	for reporterIndex := range p.rateLimit {
		slog.Info("[reporter pool] start reporter",
			"reporterIndex", reporterIndex)
		switch p.workMode {
		case "rest":
			reporter = NewReporter(p.wg, reporterIndex,
				p.metricsChan, p.serverAddress,
				p.key, setRealIP, encoder)
		case "grpc":
			reporter = NewGrpcReporter(p.wg, reporterIndex,
				p.metricsChan, p.grpcServerAddress)
		default:
			return fmt.Errorf("wrong work mode: %s (rest or grpc expected)", p.workMode)
		}
		reporter.Start(ctx)
	}
	return nil
}
