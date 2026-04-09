package cmd

import (
	"fmt"
	"log/slog"

	appconf "github.com/ganasa18/go-template/config"
	"github.com/ganasa18/go-template/http/server"
	authModule "github.com/ganasa18/go-template/internal/auth"
	authHandler "github.com/ganasa18/go-template/internal/auth/handler"
	authRepository "github.com/ganasa18/go-template/internal/auth/repository"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseModule "github.com/ganasa18/go-template/internal/base"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
)

// initHTTP wires every module inside the modular monolith and returns an HTTP server.
func initHTTP(cfg *appconf.Config) (*server.Server, error) {
	// --- Database ---
	db, err := appconf.NewDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("database init: %w", err)
	}
	slog.Info("database connected", slog.String("name", cfg.DBName))

	// --- Redis (only required for stateful JWT mode) ---
	var authSvc authService.Authenticator
	if cfg.IsStateful() {
		rdb, err := appconf.NewRedis(cfg)
		if err != nil {
			return nil, fmt.Errorf("redis init: %w", err)
		}
		slog.Info("redis connected", slog.String("addr", cfg.RedisHost+":"+cfg.RedisPort))

		authRepo := authRepository.New(db)
		authSvc = authService.New(cfg, authRepo, rdb)
	} else {
		authRepo := authRepository.New(db)
		authSvc = authService.New(cfg, authRepo, nil)
	}

	baseHTTPHandler := baseHandler.NewBaseHTTPHandler(db)
	authHTTPHandler := authHandler.New(authSvc)

	modules := []appmodule.HTTPModule{
		baseModule.NewHTTPModule(baseHTTPHandler),
		authModule.NewHTTPModule(cfg, baseHTTPHandler, authHTTPHandler, authSvc),
	}

	// --- Server ---
	srv := server.New(cfg, modules)
	return srv, nil
}
