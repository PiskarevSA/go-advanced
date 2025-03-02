package server

import (
	"flag"
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/storage"
)

var serverAddress = flag.String("a", "localhost:8080", "server address")

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
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		return nil
	}

	r := MetricsRouter(s.storage)
	err := http.ListenAndServe(*serverAddress, r)
	return err
}
