// Package config loads and validates all application configuration from
// environment variables. Call InitAppConfig() once at startup; pass *Config
// into every component that needs it.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the single source of truth for runtime configuration.
// Every field has an explicit default so the app starts in a sane state
// without a fully populated .env file (useful for integration tests).
type Config struct {
	// Application
	AppEnv      string
	AppDebug    bool
	AppVersion  string
	AppName     string
	ServiceName string

	// HTTP server
	HTTPPort            string
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration
	MaxBodyBytes        int64

	// CORS
	CORSAllowedOrigins []string

	// Rate limiting
	RateLimitRPS   float64
	RateLimitBurst int

	// Database (PostgreSQL via GORM)
	DBHost            string
	DBPort            int
	DBName            string
	DBUsername        string
	DBPassword        string
	DBSchema          string
	DBSSLMode         string // "disable" | "require" | "verify-full"
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	// Redis
	RedisHost         string
	RedisPort         string
	RedisDB           int
	RedisPassword     string
	RedisDialTimeout  time.Duration
	RedisReadTimeout  time.Duration
	RedisWriteTimeout time.Duration
	RedisPoolSize     int

	// JWT
	JWTMode          string // "stateless" | "stateful"
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	// Logging
	LogLevel  string
	LogFormat string

	// External integrations
	RobotSplitURL string // URL of the robot split-percentage service
}

// IsDevelopment returns true when AppEnv == "development".
func (c *Config) IsDevelopment() bool { return c.AppEnv == "development" }

// IsProduction returns true when AppEnv == "production".
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// IsStateful returns true when JWT_MODE is "stateful".
func (c *Config) IsStateful() bool { return strings.EqualFold(c.JWTMode, "stateful") }

// InitAppConfig reads env vars and returns a validated *Config.
// It panics on missing secrets so misconfiguration is caught at startup.
func InitAppConfig() *Config {
	c := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppDebug:    getEnvBool("APP_DEBUG", false),
		AppVersion:  getEnv("APP_VERSION", "v1"),
		AppName:     getEnv("APP_NAME", "go-template"),
		ServiceName: getEnv("SERVICE_NAME", "go-template"),

		HTTPPort:            getEnv("HTTP_SERVER_PORT", "8080"),
		HTTPReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 30*time.Second),
		HTTPWriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		HTTPIdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		HTTPShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		MaxBodyBytes:        getEnvInt64("MAX_BODY_BYTES", 4<<20), // 4 MB

		CORSAllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", nil),

		RateLimitRPS:   getEnvFloat64("RATE_LIMIT_RPS", 20),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 50),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBName:     mustEnv("DB_NAME"),
		DBUsername: mustEnv("DB_USERNAME"),
		DBPassword: mustEnv("DB_PASSWORD"),
		DBSchema:   getEnv("DB_SCHEMA", "public"),
		DBSSLMode: func() string {
			if getEnvBool("DB_SSL", false) {
				return "require"
			}
			return "disable"
		}(),
		DBMaxOpenConns:    getEnvInt("DB_MAX_CONNECTION", 100),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNECTION", 20),
		DBConnMaxLifetime: getEnvDuration("DB_MAX_LIFETIME_CONNECTION", 10*time.Minute),
		DBConnMaxIdleTime: getEnvDuration("DB_MAX_IDLE_TIME", 5*time.Minute),

		RedisHost:         getEnv("REDIS_HOST", "127.0.0.1"),
		RedisPort:         getEnv("REDIS_PORT", "6379"),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		RedisReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		RedisWriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		RedisPoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),

		JWTMode:          getEnv("JWT_MODE", "stateful"),
		JWTAccessSecret:  mustEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		JWTAccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 168*time.Hour),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "text"),

		RobotSplitURL: getEnv("ROBOT_SPLIT_URL", ""),
	}

	// Stateful mode requires a separate refresh secret.
	if c.IsStateful() && c.JWTRefreshSecret == "" {
		panic("JWT_REFRESH_SECRET is required when JWT_MODE=stateful")
	}

	return c
}

// --- helpers ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvFloat64(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvStringSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			result = append(result, s)
		}
	}
	return result
}
