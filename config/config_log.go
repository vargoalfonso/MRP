package config

import (
	"github.com/ganasa18/go-template/pkg/logger"
)

// InitLogger initialises the global slog logger from *Config.
// Call this once at startup, before any log calls are made.
func InitLogger(cfg *Config) {
	logger.New(cfg.LogLevel, cfg.LogFormat)
}
