package supplieritem

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	supplierItemHandler "github.com/ganasa18/go-template/internal/supplier_item/handler"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *supplierItemHandler.HTTPHandler
	authenticator authService.Authenticator
}

func NewHTTPModule(cfg *config.Config, base *baseHandler.BaseHTTPHandler, handler *supplierItemHandler.HTTPHandler, authenticator authService.Authenticator) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	group := v1.Group("/supplier-items")
	group.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		group.POST("", m.base.RunAction(m.handler.Create))
		group.GET("", m.base.RunAction(m.handler.List))
		group.GET("/:id", m.base.RunAction(m.handler.GetByID))
		group.PUT("/:id", m.base.RunAction(m.handler.Update))
		group.DELETE("/:id", m.base.RunAction(m.handler.Delete))
	}
}
