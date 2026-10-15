package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	defaultPort            = "8081"
	defaultDatabaseURL     = "postgres://marketplace:marketplace@localhost:5433/marketplace_user?sslmode=disable"
	defaultJWTSecret       = "local-development-jwt-secret-change-me"
	defaultJWTIssuer       = "marketplace-user-service"
	defaultAccessTokenTTL  = 20 * time.Minute
	defaultRefreshTokenTTL = 14 * 24 * time.Hour
	defaultShutdown        = 5 * time.Second
	defaultDBTimeout       = 5 * time.Second
	defaultMaxDBConns      = int32(10)
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTIssuer        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	ShutdownTimeout  time.Duration
	DatabaseTimeout  time.Duration
	DatabaseMaxConns int32
	AdminEmail       string
	AdminPassword    string
}

func Load() (Config, error) {
	accessTTL, err := duration("ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := duration("REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
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
		AccessTokenTTL:   accessTTL,
		RefreshTokenTTL:  refreshTTL,
		ShutdownTimeout:  shutdownTimeout,
		DatabaseTimeout:  databaseTimeout,
		DatabaseMaxConns: maxConns,
		AdminEmail:       os.Getenv("ADMIN_EMAIL"),
		AdminPassword:    os.Getenv("ADMIN_PASSWORD"),
	}

	if len(config.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if config.AccessTokenTTL <= 0 || config.AccessTokenTTL > time.Duration(math.MaxInt32)*time.Second {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL must be positive and fit into int32 seconds")
	}
	if config.RefreshTokenTTL <= 0 {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must be positive")
	}
	if config.RefreshTokenTTL <= config.AccessTokenTTL {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL")
	}
	if config.DatabaseMaxConns < 1 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}
	if (config.AdminEmail == "") != (config.AdminPassword == "") {
		return Config{}, fmt.Errorf("ADMIN_EMAIL and ADMIN_PASSWORD must be set together")
	}
	if config.AdminPassword != "" && (len(config.AdminPassword) < 8 || len(config.AdminPassword) > 72) {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD must contain between 8 and 72 characters")
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
