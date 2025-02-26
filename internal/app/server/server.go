package server

import (
	"net/http"
	"strconv"
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
		if len(metricName) == 0 {
			http.NotFound(res, req)
			return
		}
		f64, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(res, "incorrect metric value",
				http.StatusBadRequest)
			return
		}
		s.storage.SetGauge(metricName, f64)
	case "counter":
		if len(metricName) == 0 {
			http.NotFound(res, req)
			return
		}
		i64, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(res, "incorrect metric value",
				http.StatusBadRequest)
			return
		}
		s.storage.IncreaseCounter(metricName, i64)
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
