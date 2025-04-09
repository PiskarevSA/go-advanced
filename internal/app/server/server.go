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
	"github.com/PiskarevSA/go-advanced/internal/storage/filestorage"
	"github.com/PiskarevSA/go-advanced/internal/storage/memstorage"
	"github.com/PiskarevSA/go-advanced/internal/storage/pgstorage"
	"github.com/PiskarevSA/go-advanced/internal/usecases"
)

type usecaseStorage interface {
	GetMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error)
	UpdateMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error)
	UpdateMetrics(ctx context.Context, metrics []entities.Metric) ([]entities.Metric, error)
	GetMetricsByTypes(ctx context.Context, gauge map[entities.MetricName]entities.Gauge,
		counter map[entities.MetricName]entities.Counter) error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}

type Server struct {
	config *Config
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

	// Wait group to ensure all goroutines finish before exiting
	var wg sync.WaitGroup

	storage := s.createStorage(ctx, &wg)
	if storage == nil {
		return false
	}
	defer storage.Close(ctx)

	usecase := s.createMetricsUsecase(storage)

	server := s.createServer(usecase)

	success := true // will be false if listener could not be started
	s.startWorkers(ctx, cancel, &wg, server, &success)

	// Wait for all goroutines to finish
	wg.Wait()
	return success
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

func (s *Server) startWorkers(ctx context.Context, cancel context.CancelFunc,
	wg *sync.WaitGroup, server *http.Server, success *bool,
) {
	s.startListener(cancel, wg, server, success)
	s.startWatchdog(ctx, wg, server)
}

func (s *Server) createStorage(ctx context.Context, wg *sync.WaitGroup,
) usecaseStorage {
	var result usecaseStorage
	if len(s.config.DatabaseDSN) > 0 {
		var err error
		result, err = pgstorage.New(ctx, s.config.DatabaseDSN)
		if err != nil {
			slog.Error("[main] create pgstorage", "error", err.Error())
			return nil
		}
		slog.Info("[main] pgstorage created")
	} else if len(s.config.FileStoragePath) > 0 {
		filestorage := filestorage.New(ctx, wg,
			s.config.StoreInterval, s.config.FileStoragePath, s.config.Restore)
		result = filestorage
		slog.Info("[main] filestorage created")
	} else {
		result = memstorage.New()
		slog.Info("[main] memstorage created")
	}
	return result
}

func (s *Server) createMetricsUsecase(storage usecaseStorage,
) *usecases.MetricsUsecase {
	return usecases.NewMetricsUsecase(storage)
}

func (s *Server) createServer(usecase *usecases.MetricsUsecase) *http.Server {
	r := handlers.NewMetricsRouter(usecase).
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
