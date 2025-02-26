package server

import (
	"fmt"
	"net/http"
	"strings"

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
	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, s.Update)
	err := http.ListenAndServe("localhost:8080", mux)
	return err
}

func (s *Server) Update(res http.ResponseWriter, req *http.Request) {
	// method be POST
	if req.Method != http.MethodPost {
		http.NotFound(res, req)
		return
	}
	// header should contains "Content-Type: text/plain"
	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(res, "supported Content-Type: text/plain",
			http.StatusBadRequest)
		return
	}
	// incorrect metric type should return http.StatusBadRequest
	tail := strings.TrimPrefix(req.URL.Path, "/update/")
	metricType, tail, _ := strings.Cut(tail, "/")
	metricName, metricValue, _ := strings.Cut(tail, "/")

	switch metricType {
	case "gauge":
		fallthrough
	case "counter":
		fmt.Println("metricType", metricType)
		fmt.Println("metricName", metricName)
		fmt.Println("metricValue", metricValue)
	case "":
		http.Error(res, "empty metric type",
			http.StatusBadRequest)
		return
	default:
		http.Error(res, "unexpected metric type",
			http.StatusBadRequest)
		return
	}
}
