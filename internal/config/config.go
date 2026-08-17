package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr, DatabaseURL, UploadDir string
	RequestTimeout                   time.Duration
	MaxUploadBytes                   int64
}

func Load() Config {
	return Config{HTTPAddr: get("HTTP_ADDR", ":8080"), DatabaseURL: get("DATABASE_URL", "postgres://gate:gate@localhost:5432/gate?sslmode=disable"), UploadDir: get("UPLOAD_DIR", "./var/uploads"), RequestTimeout: duration("REQUEST_TIMEOUT", "5s"), MaxUploadBytes: number("MAX_UPLOAD_BYTES", 10<<20)}
}
func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func duration(k, d string) time.Duration {
	v, err := time.ParseDuration(get(k, d))
	if err != nil {
		return 5 * time.Second
	}
	return v
}
func number(k string, d int64) int64 {
	v, err := strconv.ParseInt(os.Getenv(k), 10, 64)
	if err != nil {
		return d
	}
	return v
}
