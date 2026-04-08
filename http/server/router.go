package server

import (
	"github.com/ganasa18/go-template/config"
	authHandler "github.com/ganasa18/go-template/internal/auth/handler"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	"github.com/gin-gonic/gin"
)

// Handlers is the single dependency bundle passed to the server.
// When adding a new domain, add its handler here — server.go never needs to change.
type Handlers struct {
	Base          *baseHandler.BaseHTTPHandler
	Auth          *authHandler.HTTPHandler
	Authenticator authService.Authenticator

	// Tambah handler baru di sini:
	// Employee *employeeHandler.HTTPHandler
	// User     *userHandler.HTTPHandler
}

// setupRoutes registers all application routes on the Gin engine.
func setupRoutes(r *gin.Engine, cfg *config.Config, h *Handlers) {
	// ── Public ──────────────────────────────────────────────────────────────
	r.GET("/health", h.Base.HealthCheck)

	v1 := r.Group("/api/v1")

	// Auth
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", h.Base.RunAction(h.Auth.Register))
		authGroup.POST("/login", h.Base.RunAction(h.Auth.Login))

		// /refresh dan /logout hanya terdaftar di mode stateful.
		if cfg.IsStateful() {
			authGroup.POST("/refresh", h.Base.RunAction(h.Auth.Refresh))

			logoutGroup := authGroup.Group("")
			logoutGroup.Use(authMiddleware.JWTMiddleware(h.Authenticator))
			logoutGroup.POST("/logout", h.Base.RunAction(h.Auth.Logout))
		}
	}

	// ── Protected (tambah domain baru di sini) ───────────────────────────────
	// protected := v1.Group("")
	// protected.Use(authMiddleware.JWTMiddleware(h.Authenticator))
	//
	// Employee (contoh setelah tambah h.Employee):
	// protected.GET("/employees", h.Base.RunAction(h.Employee.GetAll))
	// protected.GET("/employee",  h.Base.RunAction(h.Employee.GetByID))
	// protected.POST("/employee", h.Base.RunAction(h.Employee.Register))
	// protected.PUT("/employee",  h.Base.RunAction(h.Employee.Update))
}
