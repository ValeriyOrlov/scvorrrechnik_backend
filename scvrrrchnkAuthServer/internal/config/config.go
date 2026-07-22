package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DatabaseDSN     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	var dbCfg DBConfig
	dbCfg.Host = getEnv("DB_HOST", "localhost")
	dbCfg.Port = getEnv("DB_PORT", "5432")
	dbCfg.User = getEnv("DB_USER", "postgres")
	dbCfg.Password = getEnv("DB_PASSWORD", "")
	dbCfg.DBName = getEnv("DB_NAME", "auth")
	dbCfg.SSLMode = getEnv("DB_SSLMODE", "disable")

	accessTTL, err := getDurationEnv("AUTH_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("loading AUTH_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := getDurationEnv("AUTH_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("loading REFRESH_TOKEN_TTL: %w", err)
	}
	cfg := &Config{
		Port:            getEnv("APP_PORT", "8080"),
		DatabaseDSN:     fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.Host, dbCfg.Port, dbCfg.SSLMode),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}
	cfg.Port = ":" + cfg.Port
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}
