package production

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	productionHandler "github.com/ganasa18/go-template/internal/production/handler"
	productionService "github.com/ganasa18/go-template/internal/production/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *productionHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       productionService.IProductionService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *productionHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service productionService.IProductionService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleService,
		service:       service,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")

	productionGroup := v1.Group("/production")

	// 🔐 wajib login
	productionGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		productionGroup.POST("/scan-qr", roleMiddleware.RequirePermission(m.roleService, "production", "view"), m.base.RunAction(m.handler.ScanQR))

		// 🔥 SCAN IN → START PRODUCTION
		productionGroup.POST("/scan-in", roleMiddleware.RequirePermission(m.roleService, "production", "create"), m.base.RunAction(m.handler.ScanIn))

		// 🔥 SCAN OUT → COMPLETE PRODUCTION
		productionGroup.POST("/scan-out", roleMiddleware.RequirePermission(m.roleService, "production", "update"), m.base.RunAction(m.handler.ScanOut))
	}
}
