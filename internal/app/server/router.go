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
	r.Post(`/update/`, handlers.UpdateFromJSON(usecase))
	r.Post(`/update/{type}/{name}/{value}`, handlers.UpdateFromURL(usecase))
	r.Post(`/value/`, handlers.GetAsJSON(usecase))
	r.Get(`/value/{type}/{name}`, handlers.GetAsText(usecase))
	return r
}
