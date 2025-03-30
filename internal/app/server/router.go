package server

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/go-chi/chi/v5"
)

type Usecase interface {
	GetMetric(metric entities.Metric) (*entities.Metric, error)
	UpdateMetric(metric entities.Metric) (*entities.Metric, error)
	DumpIterator() func() (type_ string, name string, value string, exists bool)
}

type MetricsRouter struct {
	chi.Router
	usecase Usecase
}

func NewMetricsRouter(usecase Usecase) *MetricsRouter {
	return &MetricsRouter{
		Router:  chi.NewRouter(),
		usecase: usecase,
	}
}

func (r *MetricsRouter) WithMiddleWares(middlewares ...func(http.Handler) http.Handler) *MetricsRouter {
	r.Router.With(middlewares...)
	return r
}

func (r *MetricsRouter) WithAllHandlers() *MetricsRouter {
	mainPageHandler := handlers.MainPageHandler{Dumper: r.usecase}
	r.Get(handlers.MainPagePattern, mainPageHandler.ServeHTTP)

	updateFromJSONHandler := handlers.UpdateFromJSONHandler{Updater: r.usecase}
	r.Post(handlers.UpdateFromJSONPattern, updateFromJSONHandler.ServeHTTP)

	updateFromURLHandler := handlers.UpdateFromURLHandler{Updater: r.usecase}
	r.Post(handlers.UpdateFromURLPattern, updateFromURLHandler.ServeHTTP)

	getAsJSONHandler := handlers.GetAsJSONHandler{Getter: r.usecase}
	r.Post(handlers.GetAsJSONPattern, getAsJSONHandler.ServeHTTP)

	getAsTextHandler := handlers.GetAsTextHandler{Getter: r.usecase}
	r.Get(handlers.GetAsTextPattern, getAsTextHandler.ServeHTTP)

	return r
}
