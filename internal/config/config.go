package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	InboxDB     DatabaseConfig
	InboxWorker InboxWorkerConfig
	Repository  RepositoryConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// InboxWorkerConfig holds inbox pattern worker configuration
type InboxWorkerConfig struct {
	WorkerCount  int
	BatchSize    int
	PollInterval time.Duration
	MaxRetries   int
	RetryDelay   time.Duration
}

// RepositoryConfig holds repository configuration
type RepositoryConfig struct {
	Type string // "postgres" or "mock"
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", "10s"),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", "10s"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "mitservice"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		InboxDB: DatabaseConfig{
			Host:     getEnv("INBOX_DB_HOST", "localhost"),
			Port:     getEnv("INBOX_DB_PORT", "5433"),
			User:     getEnv("INBOX_DB_USER", "postgres"),
			Password: getEnv("INBOX_DB_PASSWORD", "password"),
			DBName:   getEnv("INBOX_DB_NAME", "mitservice_inbox"),
			SSLMode:  getEnv("INBOX_DB_SSLMODE", "disable"),
		},
		InboxWorker: InboxWorkerConfig{
			WorkerCount:  getIntEnv("INBOX_WORKER_COUNT", 5),
			BatchSize:    getIntEnv("INBOX_BATCH_SIZE", 10),
			PollInterval: getDurationEnv("INBOX_POLL_INTERVAL", "1s"),
			MaxRetries:   getIntEnv("INBOX_MAX_RETRIES", 3),
			RetryDelay:   getDurationEnv("INBOX_RETRY_DELAY", "5s"),
		},
		Repository: RepositoryConfig{
			Type: getEnv("REPOSITORY_TYPE", "postgres"),
		},
	}
}

// ConnectionString returns the database connection string
func (c *DatabaseConfig) ConnectionString() string {
	return "host=" + c.Host + " port=" + c.Port + " user=" + c.User +
		" password=" + c.Password + " dbname=" + c.DBName + " sslmode=" + c.SSLMode
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return time.Second * 10
}
