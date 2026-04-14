package supplier

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	supplierHandler "github.com/ganasa18/go-template/internal/supplier/handler"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *supplierHandler.HTTPHandler
	authenticator authService.Authenticator
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *supplierHandler.HTTPHandler,
	authenticator authService.Authenticator,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	supplierGroup := v1.Group("/suppliers")
	supplierGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		supplierGroup.POST("", m.base.RunAction(m.handler.Create))
		supplierGroup.GET("", m.base.RunAction(m.handler.List))
		supplierGroup.GET("/:id", m.base.RunAction(m.handler.GetByID))
		supplierGroup.PUT("/:id", m.base.RunAction(m.handler.Update))
		supplierGroup.DELETE("/:id", m.base.RunAction(m.handler.Delete))
	}
}
