package safety_stock_parameter

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	safetyStockHandler "github.com/ganasa18/go-template/internal/safety_stock_parameter/handler"
	safetyStockService "github.com/ganasa18/go-template/internal/safety_stock_parameter/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *safetyStockHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       safetyStockService.ISafetyStockService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *safetyStockHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service safetyStockService.ISafetyStockService,
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

	safetyStockGroup := v1.Group("/safety-stock")

	// 🔐 wajib login
	safetyStockGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		safetyStockGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "view"), m.base.RunAction(m.handler.GetSafetyStocks))
		safetyStockGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "view"), m.base.RunAction(m.handler.GetSafetyStockByID))
		safetyStockGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "create"), m.base.RunAction(m.handler.CreateSafetyStock))
		safetyStockGroup.POST("/bulk", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "create"), m.base.RunAction(m.handler.BulkCreateSafetyStock))
		safetyStockGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "update"), m.base.RunAction(m.handler.UpdateSafetyStock))
		safetyStockGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "delete"), m.base.RunAction(m.handler.DeleteSafetyStock))

		// 🔥 penting
		safetyStockGroup.GET("/calculate", roleMiddleware.RequirePermission(m.roleService, "safety-stock", "view"), m.base.RunAction(m.handler.CalculateSafetyStock))
	}
}
