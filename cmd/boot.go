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
	bomModule "github.com/ganasa18/go-template/internal/billmaterial"
	bomHandler "github.com/ganasa18/go-template/internal/billmaterial/handler"
	bomRepository "github.com/ganasa18/go-template/internal/billmaterial/repository"
	bomService "github.com/ganasa18/go-template/internal/billmaterial/service"
	departementModule "github.com/ganasa18/go-template/internal/departement"
	departementHandler "github.com/ganasa18/go-template/internal/departement/handler"
	departementRepository "github.com/ganasa18/go-template/internal/departement/repository"
	departementService "github.com/ganasa18/go-template/internal/departement/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleModule "github.com/ganasa18/go-template/internal/role"
	roleHandler "github.com/ganasa18/go-template/internal/role/handler"
	roleRepository "github.com/ganasa18/go-template/internal/role/repository"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	uploadModule "github.com/ganasa18/go-template/internal/upload"
	uploadHandler "github.com/ganasa18/go-template/internal/upload/handler"
	uploadRepository "github.com/ganasa18/go-template/internal/upload/repository"
	uploadService "github.com/ganasa18/go-template/internal/upload/service"
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

	roleRepo := roleRepository.New(db)
	roleSvc := roleService.New(roleRepo)

	departementRepo := departementRepository.New(db)
	departementSvc := departementService.New(departementRepo)
	departementHTTPHandler := departementHandler.New(departementSvc)

	baseHTTPHandler := baseHandler.NewBaseHTTPHandler(db)
	authHTTPHandler := authHandler.New(authSvc)
	roleHTTPHandler := roleHandler.New(roleSvc)

	// BOM module
	bomRepo := bomRepository.New(db)
	bomSvc := bomService.New(bomRepo)
	bomHTTPHandler := bomHandler.New(bomSvc)

	// Upload module (chunked / resumable)
	uploadRepo := uploadRepository.New(db)
	uploadSvc := uploadService.New(uploadRepo, bomRepo)
	uploadHTTPHandler := uploadHandler.New(uploadSvc)

	modules := []appmodule.HTTPModule{
		baseModule.NewHTTPModule(baseHTTPHandler),
		authModule.NewHTTPModule(cfg, baseHTTPHandler, authHTTPHandler, authSvc),
		bomModule.NewHTTPModule(cfg, baseHTTPHandler, bomHTTPHandler, authSvc, roleSvc, bomSvc),
		uploadModule.NewHTTPModule(baseHTTPHandler, uploadHTTPHandler, uploadSvc),
		roleModule.NewHTTPModule(cfg, baseHTTPHandler, roleHTTPHandler, authSvc, roleSvc),
		departementModule.NewHTTPModule(cfg, baseHTTPHandler, departementHTTPHandler, authSvc, roleSvc, departementSvc),
	}

	// --- Server ---
	srv := server.New(cfg, modules)
	return srv, nil
}
