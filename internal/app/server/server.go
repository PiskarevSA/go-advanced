package server

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/logger"
	"github.com/PiskarevSA/go-advanced/internal/storage"
	"github.com/PiskarevSA/go-advanced/internal/usecases"
)

type Server struct {
	storage *storage.MemStorage
}

func NewServer() *Server {
	return &Server{
		storage: storage.NewMemStorage(),
	}
}

// run server successfully or return false immediately
func (s *Server) Run(config *Config) bool {
	usecase := usecases.NewMetrics(s.storage)
	r := MetricsRouter(usecase)
	err := http.ListenAndServe(config.ServerAddress, r)
	if err != nil {
		logger.Plain.Error(err.Error())
		return false
	}
	return true
}
