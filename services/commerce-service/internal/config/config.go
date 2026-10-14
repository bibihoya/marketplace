package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultPort           = "8080"
	defaultDatabaseURL    = "postgres://marketplace:marketplace@localhost:5432/marketplace_commerce?sslmode=disable"
	defaultJWTSecret      = "local-development-jwt-secret-change-me"
	defaultJWTIssuer      = "marketplace-user-service"
	defaultOrderRateLimit = time.Minute
	defaultShutdown       = 5 * time.Second
	defaultDBTimeout      = 5 * time.Second
	defaultMaxDBConns     = int32(10)
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTIssuer        string
	OrderRateLimit   time.Duration
	ShutdownTimeout  time.Duration
	DatabaseTimeout  time.Duration
	DatabaseMaxConns int32
}

func Load() (Config, error) {
	orderRateLimit, err := duration("ORDER_RATE_LIMIT", defaultOrderRateLimit)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := duration("SHUTDOWN_TIMEOUT", defaultShutdown)
	if err != nil {
		return Config{}, err
	}

	databaseTimeout, err := duration("DATABASE_TIMEOUT", defaultDBTimeout)
	if err != nil {
		return Config{}, err
	}

	maxConns, err := int32Value("DATABASE_MAX_CONNS", defaultMaxDBConns)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Port:             value("PORT", defaultPort),
		DatabaseURL:      value("DATABASE_URL", defaultDatabaseURL),
		JWTSecret:        value("JWT_SECRET", defaultJWTSecret),
		JWTIssuer:        value("JWT_ISSUER", defaultJWTIssuer),
		OrderRateLimit:   orderRateLimit,
		ShutdownTimeout:  shutdownTimeout,
		DatabaseTimeout:  databaseTimeout,
		DatabaseMaxConns: maxConns,
	}

	if len(config.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if config.OrderRateLimit < 0 {
		return Config{}, fmt.Errorf("ORDER_RATE_LIMIT must not be negative")
	}
	if config.DatabaseMaxConns < 1 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}

	return config, nil
}

func value(name, fallback string) string {
	if result := os.Getenv(name); result != "" {
		return result
	}

	return fallback
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}

	result, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return result, nil
}

func int32Value(name string, fallback int32) (int32, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}

	result, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return int32(result), nil
}
