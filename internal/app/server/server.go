package server

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/storage"
)

type Server struct {
	storage *storage.MemStorage
}

func NewServer() *Server {
	return &Server{
		storage: storage.NewMemStorage(),
	}
}

// run server successfully or return error to panic in the main()
func (s *Server) Run() error {
	r := MetricsRouter(s.storage)
	err := http.ListenAndServe("localhost:8080", r)
	return err
}
