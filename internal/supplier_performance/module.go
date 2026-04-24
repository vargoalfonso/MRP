package supplierperformance

import (
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	spHandler "github.com/ganasa18/go-template/internal/supplier_performance/handler"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base          *baseHandler.BaseHTTPHandler
	handler       *spHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	handler *spHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleService,
	}
}

// RegisterRoutes registers supplier performance endpoints under /api/v1/suppliers/performance.
//
//	GET    /api/v1/suppliers/performance                       list table
//	GET    /api/v1/suppliers/performance/summary               summary cards
//	GET    /api/v1/suppliers/performance/export                export dataset
//	POST   /api/v1/suppliers/performance/:supplier_id/override override grade
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	g := v1.Group("/suppliers/performance")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.GET("", roleMiddleware.RequirePermission(m.roleService, "supplier_performance", "view"), m.base.RunAction(m.handler.List))
		g.GET("/summary", roleMiddleware.RequirePermission(m.roleService, "supplier_performance", "view"), m.base.RunAction(m.handler.Summary))
		g.GET("/charts", roleMiddleware.RequirePermission(m.roleService, "supplier_performance", "view"), m.base.RunAction(m.handler.Charts))
		g.GET("/export", roleMiddleware.RequirePermission(m.roleService, "supplier_performance", "view"), m.base.RunAction(m.handler.Export))
		g.POST("/:supplier_id/override", roleMiddleware.RequirePermission(m.roleService, "supplier_performance", "update"), m.base.RunAction(m.handler.Override))
	}
}
