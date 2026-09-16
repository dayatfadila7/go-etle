package main

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ETLEBase   string
	WorkerCount int
	QueueSize  int
	MaxRetry        int
	MaxDelayMinutes int
	MediaDir        string
	RetentionDays   int
	Port            int
	PublicURL       string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func LoadConfig() *Config {
	return &Config{
		DBHost:     getenv("DB_HOST", "localhost"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBUser:     getenv("DB_USERNAME", "ursa"),
		DBPassword: getenv("DB_PASSWORD", ""),
		DBName:     getenv("DB_NAME", "etle"),
		ETLEBase:   getenv("ETLE_BASE_URL", "https://api-etle.polri.go.id"),
		WorkerCount: getenvInt("WORKER_COUNT", 8),
		QueueSize:  getenvInt("JOB_QUEUE_SIZE", 1000),
		MaxRetry:        getenvInt("MAX_RETRY", 5),
		MaxDelayMinutes: getenvInt("MAX_DELAY_MINUTES", 3),
		MediaDir:        getenv("MEDIA_DIR", "./storage"),
		RetentionDays:   getenvInt("RETENTION_DAYS", 2),
		Port:            getenvInt("PORT", 8080),
		PublicURL:       getenv("ETLE_PUBLIC_URL", getenv("BFF_URL", "http://localhost:8080")),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		c.DBUser, c.DBPassword, c.DBName, c.DBHost, c.DBPort)
}
