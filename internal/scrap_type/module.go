package scraptype

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	scrapHandler "github.com/ganasa18/go-template/internal/scrap_type/handler"
	scrapService "github.com/ganasa18/go-template/internal/scrap_type/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *scrapHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       scrapService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *scrapHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc scrapService.IService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleSvc,
		service:       svc,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	v1 := r.Group("/api/v1")
	{
		g := v1.Group("/scrap-types")
		g.Use(auth)
		g.GET("", perm("scrap_type", "view"), m.base.RunAction(m.handler.GetScrapTypes))
		g.POST("", perm("scrap_type", "create"), m.base.RunAction(m.handler.CreateScrapType))
		g.GET("/:id", perm("scrap_type", "view"), m.base.RunAction(m.handler.GetScrapTypeByID))
		g.PUT("/:id", perm("scrap_type", "update"), m.base.RunAction(m.handler.UpdateScrapType))
		g.DELETE("/:id", perm("scrap_type", "delete"), m.base.RunAction(m.handler.DeleteScrapType))
	}
}
