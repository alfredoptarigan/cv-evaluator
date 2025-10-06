package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Qdrant   QdrantConfig
	Gemini   GeminiConfig
	Storage  StorageConfig
	Worker   WorkerConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type QdrantConfig struct {
	URL        string
	APIKey     string
	Collection string
}

type GeminiConfig struct {
	APIKey string
}

type StorageConfig struct {
	UploadPath  string
	MaxFileSize int64
}

type WorkerConfig struct {
	Concurrency       int
	RetryMaxAttempts  int
	RetryInitialDelay time.Duration
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using default values.")
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "3000"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "ai_cv_evaluator"),
		},
		Qdrant: QdrantConfig{
			URL:        getEnv("QDRANT_URL", "http://localhost:6333"),
			APIKey:     getEnv("QDRANT_API_KEY", ""),
			Collection: getEnv("QDRANT_COLLECTION", "cv_evaluator_docs"),
		},
		Gemini: GeminiConfig{
			APIKey: getEnv("GEMINI_API_KEY", ""),
		},
		Storage: StorageConfig{
			UploadPath:  getEnv("UPLOAD_PATH", "./uploads"),
			MaxFileSize: getEnvAsInt64("MAX_FILE_SIZE", 10485760),
		},
		Worker: WorkerConfig{
			Concurrency:       getEnvAsInt("WORKER_CONCURRENCY", 3),
			RetryMaxAttempts:  getEnvAsInt("RETRY_MAX_ATTEMPTS", 3),
			RetryInitialDelay: getEnvAsDuration("RETRY_INITIAL_DELAY", "2s"),
		},
	}
}

func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	valueStr := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(valueStr); err == nil {
		return duration
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}
