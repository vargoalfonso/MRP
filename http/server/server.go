package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/ganasa18/go-template/config"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/ganasa18/go-template/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// Server bundles the http.Server and its underlying Gin engine.
type Server struct {
	httpServer *http.Server
}

// New builds the Gin engine with the global middleware stack, registers all
// module routes, then wraps it in an http.Server with configured timeouts.
func New(cfg *config.Config, modules []appmodule.HTTPModule) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	if os.Getenv("DEV_SHOW_ROUTE") != "true" || cfg.IsProduction() {
		gin.DebugPrintRouteFunc = func(_, _, _ string, _ int) {}
	} else {
		gin.DebugPrintRouteFunc = func(method, path, handler string, n int) {
			fmt.Printf("Route: %-6s %-30s --> %s (%d handlers)\n",
				method, path, handler[strings.LastIndex(handler, "/")+1:], n)
		}
	}

	r := gin.New()

	// ── Global middleware ────────────────────────────────────────────────────
	r.Use(middleware.Recovery())
	r.Use(middleware.Security())
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst).Middleware())
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxBodyBytes)
		c.Next()
	})

	// ── Routes (registered by each module) ──────────────────────────────────
	setupRoutes(r, modules)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
			Handler:      r,
			ReadTimeout:  cfg.HTTPReadTimeout,
			WriteTimeout: cfg.HTTPWriteTimeout,
			IdleTimeout:  cfg.HTTPIdleTimeout,
		},
	}
}

// Start begins listening. Returns http.ErrServerClosed on graceful shutdown.
func (s *Server) Start() error {
	slog.Info("HTTP server starting", slog.String("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

// Shutdown drains active connections then stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("HTTP server shutting down")
	return s.httpServer.Shutdown(ctx)
}
