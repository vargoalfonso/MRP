package cmd

import (
	"fmt"
	"log/slog"

	appconf "github.com/ganasa18/go-template/config"
	authHandler "github.com/ganasa18/go-template/internal/auth/handler"
	authRepository "github.com/ganasa18/go-template/internal/auth/repository"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	"github.com/ganasa18/go-template/http/server"
)

// initHTTP wires all dependencies and starts the HTTP server.
// The *server.Server is returned so the caller can call Shutdown on signal.
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

	// --- Handlers ---
	// Tambah handler domain baru ke Handlers struct di http/server/router.go,
	// lalu wire di sini — server.go tidak perlu disentuh.
	h := &server.Handlers{
		Base:          baseHandler.NewBaseHTTPHandler(db),
		Auth:          authHandler.New(authSvc),
		Authenticator: authSvc,

		// Employee: employeeHandler.NewHTTPHandler(employeeSvc),
	}

	// --- Server ---
	srv := server.New(cfg, h)
	return srv, nil
}
