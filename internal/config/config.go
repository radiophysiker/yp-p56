package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	RunAddress           string        `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	DatabaseURI          string        `env:"DATABASE_URI"`
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"`
	JWTSecretKey         string        `env:"JWT_SECRET_KEY" envDefault:"secret-key-for-jwt-signing"`
	ShutdownTimeout      time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"5s"`
}

var cfg Config

func LoadConfig() (*Config, error) {
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "address and port to run service")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "database connection address")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address")
	flag.StringVar(&cfg.JWTSecretKey, "j", cfg.JWTSecretKey, "JWT secret key")
	flag.DurationVar(&cfg.ShutdownTimeout, "t", cfg.ShutdownTimeout, "server shutdown timeout")
	flag.Parse()

	return &cfg, nil
}
