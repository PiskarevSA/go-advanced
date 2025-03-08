package server

import (
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/go-chi/chi/v5"
)

type Usecase interface {
	Update(metricType string, metricName string, metricValue string) error
	Get(metricType string, metricName string) (value string, err error)
	Dump() (gauge map[string]float64, counter map[string]int64)
}

func MetricsRouter(usecase Usecase) chi.Router {
	r := chi.NewRouter()
	r.Get(`/`, handlers.MainPage(usecase))
	r.Post(`/update/{type}/{name}/{value}`, handlers.Update(usecase))
	r.Get(`/value/{type}/{name}`, handlers.Get(usecase))
	return r
}
