package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config menyimpan semua konfigurasi aplikasi yang dibaca dari environment variables.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	APIKey   string
	LogLevel string
}

type AppConfig struct {
	Env  string
	Port string
}

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

// DSN membangun PostgreSQL connection string (lib/pq format).
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// Load membaca .env lalu mem-parse semua variabel ke struct Config.
func Load() (*Config, error) {
	// Di production (Docker), env sudah di-inject langsung via --env-file.
	// godotenv.Load() hanya untuk development lokal.
	_ = godotenv.Load()

	maxOpen, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdle, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))

	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:         mustGetEnv("DB_HOST"),
			Port:         getEnv("DB_PORT", "5432"),
			User:         mustGetEnv("DB_USER"),
			Password:     mustGetEnv("DB_PASSWORD"),
			Name:         mustGetEnv("DB_NAME"),
			SSLMode:      getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns: maxOpen,
			MaxIdleConns: maxIdle,
		},
		JWT: JWTConfig{
			Secret:      mustGetEnv("JWT_SECRET"),
			ExpiryHours: jwtExpiry,
		},
		APIKey:   mustGetEnv("API_KEY"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("environment variable %q is required but not set", key))
	}
	return val
}
