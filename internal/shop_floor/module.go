package shopfloor

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	shopFloorHandler "github.com/ganasa18/go-template/internal/shop_floor/handler"
	shopFloorService "github.com/ganasa18/go-template/internal/shop_floor/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *shopFloorHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       shopFloorService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *shopFloorHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc shopFloorService.IService,
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

	g := r.Group("/api/v1/shop-floor")
	g.Use(auth)

	g.GET("/live-production/summary", perm("shop_floor", "view"), m.base.RunAction(m.handler.GetLiveProductionSummary))
	g.GET("/delivery-readiness/summary", perm("shop_floor", "view"), m.base.RunAction(m.handler.GetDeliveryReadinessSummary))
	g.GET("/production-issues/summary", perm("shop_floor", "view"), m.base.RunAction(m.handler.GetProductionIssuesSummary))
	g.GET("/scan-events/summary", perm("shop_floor", "view"), m.base.RunAction(m.handler.GetScanEventsSummary))
}
