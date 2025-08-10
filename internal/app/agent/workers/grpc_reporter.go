package workers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	"github.com/PiskarevSA/go-advanced/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcReporter struct {
	wg                *sync.WaitGroup
	index             int
	metricsChan       <-chan metrics.Metrics
	grpcServerAddress string
	client            proto.MetricsServiceClient
}

func NewGrpcReporter(
	wg *sync.WaitGroup, index int, metricsChan <-chan metrics.Metrics,
	grpcServerAddress string,
) *GrpcReporter {
	return &GrpcReporter{
		wg:                wg,
		index:             index,
		metricsChan:       metricsChan,
		grpcServerAddress: grpcServerAddress,
	}
}

func (r *GrpcReporter) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		conn, err := r.createConnection()
		if err != nil {
			slog.Info("[grpc reporter] stopping",
				"index", r.index,
				"reason", fmt.Errorf("create connection: %w", err))
			return
		}
		defer conn.Close()

		r.client = proto.NewMetricsServiceClient(conn)

		for {
			select {
			case <-ctx.Done():
				slog.Info("[grpc reporter] stopping",
					"index", r.index,
					"reason", ctx.Err())
				return
			case metrics, ok := <-r.metricsChan:
				if !ok {
					slog.Info("[grpc reporter] stopping",
						"index", r.index,
						"reason", "metrics channel closed")
					return
				}
				if err := r.report(ctx, metrics.Gauge, metrics.Counter); err != nil {
					slog.Error("[grpc reporter] report failed",
						"index", r.index,
						"error", err)
				} else {
					slog.Info("[grpc reporter] report succeeded",
						"index", r.index)
				}
			}
		}
	}()
}

func (r *GrpcReporter) createConnection() (*grpc.ClientConn, error) {
	return grpc.NewClient(
		r.grpcServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (r *GrpcReporter) report(ctx context.Context,
	gauge map[string]metrics.Gauge,
	counter map[string]metrics.Counter,
) error {
	var req proto.UpdateMetricsRequest
	req.Metrics = make([]*proto.Metric, 0, len(gauge)+len(counter))
	for key, gauge := range gauge {
		value := float64(gauge)
		m := proto.Metric{
			Metric: &proto.Metric_Gauge{
				Gauge: &proto.Gauge{
					Name:  key,
					Value: value,
				},
			},
		}
		req.Metrics = append(req.Metrics, &m)
	}

	for key, counter := range counter {
		delta := int64(counter)
		m := proto.Metric{
			Metric: &proto.Metric_Counter{
				Counter: &proto.Counter{
					Name:  key,
					Value: delta,
				},
			},
		}
		req.Metrics = append(req.Metrics, &m)
	}

	_, err := r.client.UpdateMetrics(ctx, &req)
	if err != nil {
		return fmt.Errorf("grpc UpdateMetrics: %w", err)
	}
	return nil
}
