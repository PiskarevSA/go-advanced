package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/PiskarevSA/go-advanced/internal/middleware"
	"github.com/PiskarevSA/go-advanced/internal/storage"
	"github.com/PiskarevSA/go-advanced/internal/usecases"
)

type Server struct {
	storage *storage.MemStorage
	usecase *usecases.MetricsUsecase
	config  *Config
}

func NewServer(config *Config) *Server {
	storage := storage.NewMemStorage()
	return &Server{
		storage: storage,
		usecase: usecases.NewMetricsUsecase(storage),
		config:  config,
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

	s.loadMetrics()

	server := s.createServer()

	success := true
	s.startListener(cancel, &wg, server, &success)

	s.startWatchdog(ctx, &wg, server)

	if s.config.StoreInterval > 0 {
		s.startPreserver(ctx, &wg)
	} else {
		s.usecase.OnChange = func() { s.storeMetrics("on change") }
	}

	// Wait for all goroutines to finish
	wg.Wait()
	return success
}

func (s *Server) loadMetrics() {
	if !s.config.Restore {
		slog.Info("[main] metrics file loading skipped", "path", s.config.FileStoragePath)
		return
	}
	file, err := os.Open(s.config.FileStoragePath)
	if err != nil {
		slog.Error("[main] open metrics file", "error", err.Error())
		return
	}
	defer file.Close()
	err = s.usecase.LoadMetrics(file)
	if err != nil {
		slog.Error("[main] load metrics file", "error", err.Error())
		return
	}
	slog.Info("[main] metrics file loaded", "path", s.config.FileStoragePath)
}

func (s *Server) storeMetrics(caller string) {
	file, err := os.Create(s.config.FileStoragePath)
	if err != nil {
		msg := fmt.Sprintf("[%v] create metrics file", caller)
		slog.Error(msg, "error", err.Error())
		return
	}
	defer file.Close()
	err = s.usecase.StoreMetrics(file)
	if err != nil {
		msg := fmt.Sprintf("[%v] store metrics file", caller)
		slog.Error(msg, "error", err.Error())
		return
	}

	msg := fmt.Sprintf("[%v] metrics file stored", caller)
	slog.Info(msg, "path", s.config.FileStoragePath)
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

func (s *Server) startPreserver(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	storeInterval := time.Duration(s.config.StoreInterval) * time.Second
	go func() {
		defer wg.Done()
		slog.Info("[preserver] start")
		for {
			s.storeMetrics("preserver")
			// sleep storeInterval or interrupt
			for t := time.Duration(0); t < storeInterval; t += updateInterval {
				select {
				case <-ctx.Done():
					// Handle context cancellation (graceful shutdown)
					slog.Info("[preserver] stopping", "error", ctx.Err())
					s.storeMetrics("preserver") // save changes
					return
				default:
					time.Sleep(updateInterval)
				}
			}
		}
	}()
}
