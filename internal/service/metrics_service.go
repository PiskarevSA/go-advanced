package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const timeout = 15 * time.Second

type metricsUsecase interface {
	UpdateMetrics(ctx context.Context, metrics []entities.Metric) ([]entities.Metric, error)
}

// *MetricsService implements proto.MetricsServiceServer
var _ proto.MetricsServiceServer = (*MetricsService)(nil)

type MetricsService struct {
	proto.UnimplementedMetricsServiceServer
	metricsUsecase metricsUsecase
}

func NewMetricsService(metricsUsecase metricsUsecase) *MetricsService {
	return &MetricsService{
		metricsUsecase: metricsUsecase,
	}
}

func convertBatchMetricFromUpdateMetricsRequest(req *proto.UpdateMetricsRequest,
) ([]entities.Metric, error) {
	var result []entities.Metric
	for i, metric := range req.Metrics {
		gauge := metric.GetGauge()
		counter := metric.GetCounter()

		if gauge != nil {
			if len(gauge.Name) == 0 {
				return nil, fmt.Errorf("metric[%v]: %w", i, entities.ErrEmptyMetricName)
			}
			entityMetric := entities.Metric{
				Type:  entities.MetricTypeGauge,
				Name:  entities.MetricName(gauge.Name),
				Value: entities.Gauge(gauge.Value),
			}
			result = append(result, entityMetric)
		} else if counter != nil {
			if len(counter.Name) == 0 {
				return nil, fmt.Errorf("metric[%v]: %w", i, entities.ErrEmptyMetricName)
			}
			entityMetric := entities.Metric{
				Type:  entities.MetricTypeCounter,
				Name:  entities.MetricName(counter.Name),
				Delta: entities.Counter(counter.Value),
			}
			result = append(result, entityMetric)
		} else {
			return nil, fmt.Errorf("metric[%v]: %w", i, entities.ErrEmptyMetricType)
		}
	}
	return result, nil
}

func convertEntityMetric(metric entities.Metric) (*proto.Metric, error) {
	var result proto.Metric
	switch metric.Type {
	case entities.MetricTypeGauge:
		result.Metric = &proto.Metric_Gauge{
			Gauge: &proto.Gauge{
				Name:  string(metric.Name),
				Value: float64(metric.Value),
			},
		}
	case entities.MetricTypeCounter:
		result.Metric = &proto.Metric_Counter{
			Counter: &proto.Counter{
				Name:  string(metric.Name),
				Value: int64(metric.Delta),
			},
		}
	default:
		return nil, entities.NewInternalError(
			"unexpected internal metric type: "+metric.Type.String(), nil)
	}
	return &result, nil
}

func convertEntityMetrics(metrics []entities.Metric) ([]*proto.Metric, error) {
	result := make([]*proto.Metric, 0)
	for i, entityMetric := range metrics {
		metric, err := convertEntityMetric(entityMetric)
		if err != nil {
			return nil, fmt.Errorf("metric[%v]: %w", i, err)
		}
		result = append(result, metric)
	}
	return result, nil
}

func (s *MetricsService) UpdateMetrics(
	ctx context.Context, req *proto.UpdateMetricsRequest,
) (*proto.UpdateMetricsResponse, error) {
	validMetrics, err := convertBatchMetricFromUpdateMetricsRequest(req)
	if err != nil {
		return nil, toGrpcError(fmt.Errorf("convert to entities: %w", err))
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	updatedMetrics, err := s.metricsUsecase.UpdateMetrics(ctx, validMetrics)
	if err != nil {
		return nil, toGrpcError(fmt.Errorf("update metrics: %w", err))
	}
	// success
	var response proto.UpdateMetricsResponse
	response.Metrics, err = convertEntityMetrics(updatedMetrics)
	if err != nil {
		return nil, toGrpcError(fmt.Errorf("convert from entities: %w", err))
	}

	return &response, nil
}

func toGrpcError(err error) error {
	defer slog.Error("update error handled", "error", err)

	var internalError *entities.InternalError
	// incorrect metric type should return http.StatusBadRequest
	switch {
	case errors.Is(err, entities.ErrEmptyMetricType): // +
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, entities.ErrEmptyMetricName): // +
		return status.Error(codes.NotFound, err.Error())
	case errors.As(err, &internalError): // +
		return status.Error(codes.Internal, err.Error())
	default:
		// unexpected error
		return status.Error(codes.Internal, err.Error())
	}
}
