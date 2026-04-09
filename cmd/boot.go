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
	customerModule "github.com/ganasa18/go-template/internal/customer"
	customerHandler "github.com/ganasa18/go-template/internal/customer/handler"
	customerRepository "github.com/ganasa18/go-template/internal/customer/repository"
	customerService "github.com/ganasa18/go-template/internal/customer/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	supplierModule "github.com/ganasa18/go-template/internal/supplier"
	supplierHandler "github.com/ganasa18/go-template/internal/supplier/handler"
	supplierRepository "github.com/ganasa18/go-template/internal/supplier/repository"
	supplierService "github.com/ganasa18/go-template/internal/supplier/service"
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
	customerRepo := customerRepository.New(db)
	customerSvc := customerService.New(customerRepo)
	customerHTTPHandler := customerHandler.New(customerSvc)
	supplierRepo := supplierRepository.New(db)
	supplierSvc := supplierService.New(supplierRepo)
	supplierHTTPHandler := supplierHandler.New(supplierSvc)

	modules := []appmodule.HTTPModule{
		baseModule.NewHTTPModule(baseHTTPHandler),
		authModule.NewHTTPModule(cfg, baseHTTPHandler, authHTTPHandler, authSvc),
		customerModule.NewHTTPModule(cfg, baseHTTPHandler, customerHTTPHandler, authSvc),
		supplierModule.NewHTTPModule(cfg, baseHTTPHandler, supplierHTTPHandler, authSvc),
	}

	// --- Server ---
	srv := server.New(cfg, modules)
	return srv, nil
}
