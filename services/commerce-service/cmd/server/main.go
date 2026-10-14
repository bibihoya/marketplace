package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"marketplace/commerce-service/internal/auth"
	"marketplace/commerce-service/internal/config"
	"marketplace/commerce-service/internal/handler"
	"marketplace/commerce-service/internal/repository"
	"marketplace/commerce-service/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration is invalid", "error", err)
		os.Exit(1)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database configuration is invalid", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = cfg.DatabaseMaxConns

	databaseCtx, cancelDatabase := context.WithTimeout(context.Background(), cfg.DatabaseTimeout)
	defer cancelDatabase()
	pool, err := pgxpool.NewWithConfig(databaseCtx, poolConfig)
	if err != nil {
		logger.Error("database pool initialization failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(databaseCtx); err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	postgresRepository := repository.NewPostgres(pool)
	commerceService := service.New(postgresRepository, cfg.OrderRateLimit)
	verifier := auth.NewVerifier(cfg.JWTSecret, cfg.JWTIssuer)
	httpHandler, err := handler.NewHandler(commerceService, verifier, logger)
	if err != nil {
		logger.Error("HTTP handler initialization failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("commerce-service is listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
