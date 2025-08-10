package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/PiskarevSA/go-advanced/internal/middleware"
	rsamiddleware "github.com/PiskarevSA/go-advanced/internal/middleware/rsa"
	"github.com/PiskarevSA/go-advanced/internal/middleware/subnet"
	"github.com/PiskarevSA/go-advanced/internal/proto"
	"github.com/PiskarevSA/go-advanced/internal/service"
	"github.com/PiskarevSA/go-advanced/internal/storage/filestorage"
	"github.com/PiskarevSA/go-advanced/internal/storage/memstorage"
	"github.com/PiskarevSA/go-advanced/internal/storage/pgstorage"
	"github.com/PiskarevSA/go-advanced/internal/usecases"
	"google.golang.org/grpc"
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

	server, err := s.createServer(usecase)
	if err != nil {
		slog.Error("[main] create server", "error", err.Error())
		return false
	}

	var success atomic.Bool // will be false if any rest or grpc listener could not be started
	success.Store(true)

	s.startWorkers(ctx, cancel, &wg, server, &success)

	grpcListen, grpcServer, err := s.createGrpcServer(usecase)
	if err != nil {
		slog.Error("[main] create grpc server", "error", err.Error())
		return false
	}
	s.startGrpcWorkers(ctx, cancel, &wg, grpcListen, grpcServer, &success)

	// Wait for all goroutines to finish
	wg.Wait()
	return success.Load()
}

func (s *Server) setupSignalHandler() (context.Context, context.CancelFunc) {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Channel to listen for system signals (e.g., Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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
	wg *sync.WaitGroup, server *http.Server, success *atomic.Bool,
) {
	s.startListener(cancel, wg, server, success)
	s.startWatchdog(ctx, wg, server)
}

func (s *Server) startGrpcWorkers(ctx context.Context, cancel context.CancelFunc,
	wg *sync.WaitGroup, listen net.Listener, server *grpc.Server, success *atomic.Bool,
) {
	s.startGrpcListener(cancel, wg, listen, server, success)
	s.startGrpcWatchdog(ctx, wg, server)
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
		filestorage := filestorage.New(
			s.config.StoreInterval, s.config.FileStoragePath, s.config.Restore)
		filestorage.Init()
		filestorage.Start(ctx, wg)
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

func (s *Server) createServer(usecase *usecases.MetricsUsecase) (*http.Server, error) {
	middlewares := []func(http.Handler) http.Handler{
		middleware.Summary,
	}
	if len(s.config.TrustedSubnet) > 0 {
		verifyHeader, err := subnet.VerifyHeader(s.config.TrustedSubnet, handlers.WillModifyMetrics)
		if err != nil {
			return nil, fmt.Errorf("verify subnet header: %w", err)
		}
		if verifyHeader != nil {
			middlewares = append(middlewares, verifyHeader)
		}
	}
	if len(s.config.CryptoKey) > 0 {
		decoder, err := rsamiddleware.Decoder(s.config.CryptoKey)
		if err != nil {
			return nil, fmt.Errorf("rsamiddleware decoder: %w", err)
		}
		if decoder != nil {
			middlewares = append(middlewares, decoder)
		}
	}
	middlewares = append(middlewares,
		middleware.Integrity(s.config.Key),
		middleware.Encoding)
	r := handlers.NewMetricsRouter(usecase).
		WithMiddlewares(middlewares...).
		WithAllHandlers()
	server := http.Server{
		Addr: s.config.ServerAddress,
	}
	server.Handler = r
	return &server, nil
}

func (s *Server) createGrpcServer(usecase *usecases.MetricsUsecase,
) (net.Listener, *grpc.Server, error) {
	listen, err := net.Listen("tcp", s.config.GrpcServerAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("listen: %w", err)
	}
	server := grpc.NewServer()
	proto.RegisterMetricsServiceServer(
		server, service.NewMetricsService(usecase))

	return listen, server, nil
}

func (s *Server) startListener(cancel context.CancelFunc, wg *sync.WaitGroup,
	server *http.Server, success *atomic.Bool,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[listener] start")

		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("[listener] server.ListenAndServe() error", "error", err.Error())
			success.Store(false)

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

func (s *Server) startGrpcListener(cancel context.CancelFunc, wg *sync.WaitGroup,
	listen net.Listener, server *grpc.Server, success *atomic.Bool,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[grpc listener] start")

		if err := server.Serve(listen); !errors.Is(err, grpc.ErrServerStopped) {
			slog.Error("[grpc listener] server.ListenAndServe() error", "error", err.Error())
			success.Store(false)

			// Cancel the context to notify all goroutines to stop
			cancel()
		}
		slog.Info("[grpc listener] Stopped serving new connections.")
	}()
}

func (s *Server) startGrpcWatchdog(ctx context.Context, wg *sync.WaitGroup, server *grpc.Server) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("[grpc watchdog] start")
		<-ctx.Done()

		slog.Info("[grpc watchdog] server.GracefulStop() initiated", "reason", ctx.Err())
		server.GracefulStop()
		slog.Info("[grpc watchdog] server.GracefulStop() completed")
	}()
}
