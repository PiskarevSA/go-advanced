package server

import (
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/go-chi/chi/v5"
)

func MetricsRouter(repo handlers.Repositories) chi.Router {
	r := chi.NewRouter()
	r.Get(`/`, handlers.MainPage(repo))
	r.Post(`/update/{type}/{name}/{value}`, handlers.Update(repo))
	r.Get(`/value/{type}/{name}`, handlers.Get(repo))
	return r
}
