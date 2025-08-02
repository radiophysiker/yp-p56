package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/config"
	"github.com/radiophysiker/d56/internal/infrastructure/container"
)

func Run() error {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("cannot create logger: %w", err)
	}
	zap.ReplaceGlobals(logger)
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			logger.Error("cannot sync logger", zap.Error(err))
		}
	}(logger)

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}
	logger.Info("Loaded config", zap.Any("config", cfg))

	// Initialize dependency injection container
	diContainer, err := container.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("cannot initialize DI container: %w", err)
	}
	defer diContainer.Close()
	// Get router from DI container
	httpHandler := diContainer.GetRouter().Setup()

	// Create context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if accrualProcessorService := diContainer.GetAccrualProcessorService(); accrualProcessorService != nil {
		go accrualProcessorService.Start(ctx)
		logger.Info("Started accrual processor service")
	}

	// Start server in a goroutine
	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: httpHandler,
	}
	go func() {
		logger.Info("Starting server", zap.String("address", cfg.RunAddress))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error", zap.Error(err))
			cancel()
		}
	}()
	<-ctx.Done()
	logger.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("Server exited")
	return nil
}
