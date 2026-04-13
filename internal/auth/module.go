package auth

import (
	"github.com/ganasa18/go-template/config"
	authHandler "github.com/ganasa18/go-template/internal/auth/handler"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

// HTTPModule owns all auth route registration.
type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *authHandler.HTTPHandler
	authenticator authService.Authenticator
}

// NewHTTPModule constructs the auth HTTP module.
func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *authHandler.HTTPHandler,
	authenticator authService.Authenticator,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
	}
}

// RegisterRoutes implements module.HTTPModule.
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")

	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", m.base.RunAction(m.handler.Register))
		authGroup.POST("/login", m.base.RunAction(m.handler.Login))

		authGroup.POST("/refresh", m.base.RunAction(m.handler.Refresh))

		authGroup.POST("/set-password", m.base.RunAction(m.handler.SetPassword))

		logoutGroup := authGroup.Group("")
		logoutGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))
		logoutGroup.POST("/logout", m.base.RunAction(m.handler.Logout))

	}
}
