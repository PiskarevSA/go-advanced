package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers/adapters"
	"github.com/go-chi/chi/v5"
)

const (
	docTemplate = `<!DOCTYPE html>
<title>Metrics</title>
<body>
	<table>
		<tr>
			<th>type</th>
			<th>key</th>
			<th>value</th>
		</tr>%s
	</table>
</body>
`

	rowTemplate = `
		<tr>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
		</tr>`
)

type metricsUsecase interface {
	GetMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error)
	UpdateMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error)
	UpdateMetrics(ctx context.Context, metrics []entities.Metric) ([]entities.Metric, error)
	DumpIterator(ctx context.Context) (func() (type_ string, name string, value string, exists bool), error)
	Ping(ctx context.Context) error
}

type MetricsRouter struct {
	chi.Router
	metricsUsecase metricsUsecase
}

func NewMetricsRouter(usecase metricsUsecase) *MetricsRouter {
	return &MetricsRouter{
		Router:         chi.NewRouter(),
		metricsUsecase: usecase,
	}
}

func (r *MetricsRouter) WithMiddlewares(middlewares ...func(http.Handler) http.Handler) *MetricsRouter {
	r.Router = r.Router.With(middlewares...)
	return r
}

func (r *MetricsRouter) WithAllHandlers() *MetricsRouter {
	r.Get(`/`, r.mainPageHandler)
	r.Post(`/update/`, r.updateFromJSONHandler)
	r.Post(`/updates/`, r.updateBatchFromJSONHandler)
	r.Post(`/update/{type}/{name}/{value}`, r.updateFromURLHandler)
	r.Post(`/value/`, r.getAsJSONHandler)
	r.Get(`/value/{type}/{name}`, r.getAsTextHandler)
	r.Get(`/ping`, r.ping)

	return r
}

// mainPageHandler handles endpoint: GET /
// request: none
// response	type: "text/html", body: html document containing dumped metrics
func (r *MetricsRouter) mainPageHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	metricsIterator, err := r.metricsUsecase.DumpIterator(ctx)
	if err != nil {
		handleAsInternalServerError(err, res)
	}

	var rows string
	for {
		type_, name, value, exists := metricsIterator()
		if !exists {
			break
		}
		rows += fmt.Sprintf(rowTemplate, type_, name, value)
	}

	doc := fmt.Sprintf(docTemplate, rows)

	res.Header().Set("Content-Type", "text/html")
	_, err = res.Write([]byte(doc))
	if err != nil {
		handleAsInternalServerError(err, res)
	}
}

// getAsJSONHandler handles endpoint: POST /value/
// request type: "application/json", body: models.Metric
// response	type: "application/json", body: models.Metric
func (r *MetricsRouter) getAsJSONHandler(res http.ResponseWriter, req *http.Request) {
	validMetric, err := adapters.ConvertMetricFromGetAsJSONRequest(req)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	responseMetric, err := r.metricsUsecase.GetMetric(ctx, *validMetric)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	// success
	response, err := adapters.ConvertEntityMetric(*responseMetric)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(response); err != nil {
		handleAsInternalServerError(err, res)
		return
	}
}

// getAsTextHandler handles endpoint: GET /value/{type}/{name}
// request: none
// response type: "text/plain; charset=utf-8", body: metric value as string
func (r *MetricsRouter) getAsTextHandler(res http.ResponseWriter, req *http.Request) {
	validMetric, err := adapters.ConvertMetricFromGetGetAsTextRequest(req)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	responseMetric, err := r.metricsUsecase.GetMetric(ctx, *validMetric)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	// success
	var response string
	switch responseMetric.Type {
	case entities.MetricTypeGauge:
		response = fmt.Sprint(responseMetric.Value)
	case entities.MetricTypeCounter:
		response = fmt.Sprint(responseMetric.Delta)
	default:
		err := entities.NewInternalError(
			"unexpected internal metric type: " + responseMetric.Type.String())
		handleGetterError(err, res, req)
		return
	}
	_, err = res.Write([]byte(response))
	if err != nil {
		err := entities.NewInternalError(
			"response writing error: " + responseMetric.Type.String())
		handleGetterError(err, res, req)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func handleGetterError(err error, res http.ResponseWriter, req *http.Request) {
	var (
		invalidMetricTypeError  *entities.InvalidMetricTypeError
		metricNameNotFoundError *entities.MetricNameNotFoundError
		jsonRequestDecodeError  *entities.JSONRequestDecodeError
		internalError           *entities.InternalError
	)
	switch {
	case errors.Is(err, entities.ErrJSONRequestExpected):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrEmptyMetricName):
		http.NotFound(res, req)
	case errors.As(err, &invalidMetricTypeError):
		http.NotFound(res, req)
	case errors.As(err, &metricNameNotFoundError):
		http.NotFound(res, req)
	case errors.As(err, &jsonRequestDecodeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.As(err, &internalError):
		http.Error(res, err.Error(), http.StatusInternalServerError)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
	slog.Error("getter error handled", "error", err)
}

// updateFromJSONHandler handles endpoint: POST /update/
// request type: "application/json", body: models.Metric
// response	type: "application/json", body: models.Metric
func (r *MetricsRouter) updateFromJSONHandler(res http.ResponseWriter, req *http.Request) {
	validMetric, err := adapters.ConvertMetricFromUpdateFromJSONRequest(req)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updatedMetric, err := r.metricsUsecase.UpdateMetric(ctx, *validMetric)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}
	// success
	response, err := adapters.ConvertEntityMetric(*updatedMetric)
	if err != nil {
		handleUpdateError(err, res, req)
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(response); err != nil {
		handleAsInternalServerError(err, res)
		return
	}
}

// updateBatchFromJSONHandler handles endpoint: POST /updates/
// request type: "application/json", body: []models.Metric
// response type: "application/json", body: []models.Metric
func (r *MetricsRouter) updateBatchFromJSONHandler(res http.ResponseWriter, req *http.Request) {
	validMetrics, err := adapters.ConvertBatchMetricFromUpdateFromJSONRequest(req)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updatedMetrics, err := r.metricsUsecase.UpdateMetrics(ctx, validMetrics)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}
	// success
	response, err := adapters.ConvertEntityMetrics(updatedMetrics)
	if err != nil {
		handleUpdateError(err, res, req)
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(response); err != nil {
		handleAsInternalServerError(err, res)
		return
	}
}

// updateFromURLHandler handles endpoint: POST /update/{type}/{name}/{value}
// request: none
// response	type: "text/plain; charset=utf-8", body: none
func (r *MetricsRouter) updateFromURLHandler(res http.ResponseWriter, req *http.Request) {
	validMetric, err := adapters.ConvertMetricFromUpdateFromURLRequest(req)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := r.metricsUsecase.UpdateMetric(ctx, *validMetric); err != nil {
		handleUpdateError(err, res, req)
		return
	}
	// success
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func handleUpdateError(err error, res http.ResponseWriter, req *http.Request) {
	var (
		invalidMetricTypeError     *entities.InvalidMetricTypeError
		metricValueIsNotValidError *entities.MetricValueIsNotValidError
		jsonRequestDecodeError     *entities.JSONRequestDecodeError
		internalError              *entities.InternalError
	)
	// incorrect metric type should return http.StatusBadRequest
	switch {
	case errors.Is(err, entities.ErrJSONRequestExpected):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.As(err, &invalidMetricTypeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrEmptyMetricName):
		http.NotFound(res, req)
	case errors.As(err, &metricValueIsNotValidError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrMissingValue):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrMissingDelta):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.As(err, &jsonRequestDecodeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.As(err, &internalError):
		http.Error(res, err.Error(), http.StatusInternalServerError)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
	slog.Error("update error handled", "error", err)
}

func (r *MetricsRouter) ping(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.metricsUsecase.Ping(ctx); err != nil {
		handleAsInternalServerError(err, res)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func handleAsInternalServerError(err error, res http.ResponseWriter) {
	slog.Error("internal error handled", "error", err)
	http.Error(res, err.Error(), http.StatusInternalServerError)
}
