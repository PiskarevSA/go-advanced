package server

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/go-chi/chi/v5"
)

type Usecase interface {
	handlers.Getter
	handlers.Updater
	handlers.Dumper
}

func MetricsRouter(usecase Usecase, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter().With(middlewares...)
	r.Get(`/`, handlers.MainPage(usecase))
	r.Post(`/update/`, handlers.UpdateJSON(usecase))
	r.Post(`/update/{type}/{name}/{value}`, handlers.Update(usecase))
	r.Post(`/value/`, handlers.GetJSON(usecase))
	r.Get(`/value/{type}/{name}`, handlers.Get(usecase))
	return r
}
