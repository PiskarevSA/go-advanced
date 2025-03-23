package server

import (
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/PiskarevSA/go-advanced/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type Usecase interface {
	IsGauge(metricType string) (exists bool, err error)
	Update(metricType string, metricName string, metricValue string) error
	UpdateGauge(metricName string, value *float64) error
	IncreaseCounter(metricName string, delta *int64) (value *int64, err error)
	Get(metricType string, metricName string) (value string, err error)
	GetGauge(metricName string) (value *float64, err error)
	GetCounter(metricName string) (value *int64, err error)
	DumpIterator() func() (type_ string, name string, value string, exists bool)
}

func MetricsRouter(usecase Usecase) chi.Router {
	r := chi.NewRouter().With(middleware.Summary, middleware.Encoding)
	r.Get(`/`, handlers.MainPage(usecase))
	r.Post(`/update/`, handlers.UpdateJSON(usecase))
	r.Post(`/update/{type}/{name}/{value}`, handlers.Update(usecase))
	r.Post(`/value/`, handlers.GetJSON(usecase))
	r.Get(`/value/{type}/{name}`, handlers.Get(usecase))
	return r
}
