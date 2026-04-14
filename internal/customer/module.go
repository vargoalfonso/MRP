package customer

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	customerHandler "github.com/ganasa18/go-template/internal/customer/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *customerHandler.HTTPHandler
	authenticator authService.Authenticator
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *customerHandler.HTTPHandler,
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
	customerGroup := v1.Group("/customers")
	customerGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		customerGroup.POST("", m.base.RunAction(m.handler.Create))
		customerGroup.GET("", m.base.RunAction(m.handler.List))
		customerGroup.GET("/:id", m.base.RunAction(m.handler.GetByID))
		customerGroup.PUT("/:id", m.base.RunAction(m.handler.Update))
		customerGroup.DELETE("/:id", m.base.RunAction(m.handler.Delete))
	}
}
