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

	"github.com/PiskarevSA/go-advanced/internal/middleware"
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
	ctx, cancel := s.setupSignalHandler()
	defer cancel() // Ensure cancel is called at the end to clean up

	return s.startWorkers(ctx, cancel, config)
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
	ctx context.Context, cancel context.CancelFunc, config *Config,
) bool {
	// Wait group to ensure all goroutines finish before exiting
	var wg sync.WaitGroup

	success := true
	s.startListener(ctx, cancel, &wg, config.ServerAddress, &success)

	// Wait for all goroutines to finish
	wg.Wait()
	return success
}

func (s *Server) startListener(ctx context.Context, cancel context.CancelFunc,
	wg *sync.WaitGroup, ServerAddress string, success *bool,
) {
	usecase := usecases.NewMetrics(s.storage)
	r := MetricsRouter(usecase, middleware.Summary, middleware.Encoding)
	server := http.Server{
		Addr: ServerAddress,
	}
	server.Handler = r

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[listener] start ")

		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("[listener] server.ListenAndServe() error", "error", err.Error())
			*success = false

			// Cancel the context to notify all goroutines to stop
			cancel()
		}
		slog.Info("[listener] Stopped serving new connections.")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownRelease()

		slog.Info("[watchdog] server.Shutdown() initiated")
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("[watchdog] server.Shutdown() error", "error", err.Error())
		} else {
			slog.Info("[watchdog] server.Shutdown() completed")
		}
	}()
}
