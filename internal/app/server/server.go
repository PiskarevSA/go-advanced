package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/PiskarevSA/go-advanced/internal/middleware"
	"github.com/PiskarevSA/go-advanced/internal/storage"
	"github.com/PiskarevSA/go-advanced/internal/usecases"
)

type usecaseStorage interface {
	GetMetric(metric entities.Metric) (*entities.Metric, error)
	UpdateMetric(metric entities.Metric) (*entities.Metric, error)
	GetMetricsByTypes() (gauge map[entities.MetricName]entities.Gauge,
		counter map[entities.MetricName]entities.Counter)
	Ping() error
	Close() error
}

type Server struct {
	storage usecaseStorage
	usecase *usecases.MetricsUsecase
	config  *Config
}

func NewServer(config *Config) *Server {
	return &Server{
		config: config,
	}
}

// run server successfully or return false immediately
func (s *Server) Run() bool {
	ctx, cancel := s.setupSignalHandler()
	defer cancel() // Ensure cancel is called at the end to clean up

	return s.startWorkers(ctx, cancel)
}

func (s *Server) setupSignalHandler() (context.Context, context.CancelFunc) {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Channel to listen for system signals (e.g., Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("[signal handler] Waiting for an interrupt signal...")

		// Wait for an interrupt signal to initiate graceful shutdown
		<-sigChan

		// Handle shutdown signal (Ctrl+C or SIGTERM)
		slog.Info("[signal handler] Received shutdown signal. Shutting down gracefully...")

		// Cancel the context to notify all goroutines to stop
		cancel()
		slog.Info("[signal handler] Cancel func called")
	}()
	return ctx, cancel
}

func (s *Server) startWorkers(
	ctx context.Context, cancel context.CancelFunc,
) bool {
	// Wait group to ensure all goroutines finish before exiting
	var wg sync.WaitGroup

	s.createStorage(ctx, &wg)
	defer s.storage.Close()

	s.createMetricsUsecase()

	server := s.createServer()

	success := true
	s.startListener(cancel, &wg, server, &success)

	s.startWatchdog(ctx, &wg, server)

	// Wait for all goroutines to finish
	wg.Wait()
	return success
}

func (s *Server) createStorage(ctx context.Context, wg *sync.WaitGroup) {
	if len(s.config.DatabaseDSN) > 0 {
		s.storage = storage.NewPgStorage(s.config.DatabaseDSN)
		slog.Info("[main] pgstorage created")
	} else if len(s.config.FileStoragePath) > 0 {
		filestorage := storage.NewFileStorage(ctx, wg,
			s.config.StoreInterval, s.config.FileStoragePath, s.config.Restore)
		s.storage = filestorage
		slog.Info("[main] filestorage created")
	} else {
		s.storage = storage.NewMemStorage()
		slog.Info("[main] memstorage created")
	}
}

func (s *Server) createMetricsUsecase() {
	s.usecase = usecases.NewMetricsUsecase(s.storage)
}

func (s *Server) createServer() *http.Server {
	r := handlers.NewMetricsRouter(s.usecase).
		WithMiddlewares(middleware.Summary, middleware.Encoding).
		WithAllHandlers()
	server := http.Server{
		Addr: s.config.ServerAddress,
	}
	server.Handler = r
	return &server
}

func (s *Server) startListener(cancel context.CancelFunc, wg *sync.WaitGroup,
	server *http.Server, success *bool,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[listener] start")

		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("[listener] server.ListenAndServe() error", "error", err.Error())
			*success = false

			// Cancel the context to notify all goroutines to stop
			cancel()
		}
		slog.Info("[listener] Stopped serving new connections.")
	}()
}

func (s *Server) startWatchdog(ctx context.Context, wg *sync.WaitGroup, server *http.Server) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[watchdog] start")
		<-ctx.Done()

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownRelease()

		slog.Info("[watchdog] server.Shutdown() initiated", "reason", ctx.Err())
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("[watchdog] server.Shutdown() error", "error", err.Error())
		} else {
			slog.Info("[watchdog] server.Shutdown() completed")
		}
	}()
}
