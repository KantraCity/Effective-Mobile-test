package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DBURL        string
	LogLevel     string
	MaxIdleConns int
	MaxOpenConns int
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:         getEnv("PORT", "8080"),
		DBURL:        getEnv("DB_URL", ""),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 10),
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}
