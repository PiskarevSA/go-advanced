package server

import (
	"log"
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

// TODO PR #5
// вот это всё бы вынести в отдельный пакет и хелпер метод и делать в мэйне
// и при надобности передавать в Run или структуру server'а
//
// прим. пер.: речь при строки по работе с flag и env

// run server successfully or return false immediately
func (s *Server) Run(config *Config) bool {
	r := MetricsRouter(s.storage)
	err := http.ListenAndServe(config.ServerAddress, r)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
