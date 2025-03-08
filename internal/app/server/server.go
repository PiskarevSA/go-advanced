package server

import (
	"flag"
	"log"
	"net/http"

	"github.com/caarlos0/env/v6"

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

// run server successfully or return error to panic in the main()
func (s *Server) Run() error {
	var config Config
	// flags takes less priority according to task description
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080",
		"server address; env: ADDRESS")
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		return nil
	}
	log.Printf("config after flags: %+v\n", config)

	// enviromnent takes higher priority according to task description
	err := env.Parse(&config)
	if err != nil {
		log.Println(err)
		flag.Usage()
		return nil
	}
	log.Printf("config after env: %+v\n", config)

	r := MetricsRouter(s.storage)
	err = http.ListenAndServe(config.ServerAddress, r)
	return err
}
