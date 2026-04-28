package stockdaysparameter

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	stockdaysHandler "github.com/ganasa18/go-template/internal/stockdaysparameter/handler"
	stockdaysService "github.com/ganasa18/go-template/internal/stockdaysparameter/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *stockdaysHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       stockdaysService.IStockdayService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *stockdaysHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service stockdaysService.IStockdayService,
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

	stockdaysGroup := v1.Group("/stockdays")

	// 🔐 wajib login
	stockdaysGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		stockdaysGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "stockdays", "view"), m.base.RunAction(m.handler.GetAll))
		stockdaysGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "stockdays", "create"), m.base.RunAction(m.handler.Create))
		stockdaysGroup.POST("/bulk", roleMiddleware.RequirePermission(m.roleService, "stockdays", "create"), m.base.RunAction(m.handler.BulkCreate))
		stockdaysGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "stockdays", "view"), m.base.RunAction(m.handler.GetByID))
		stockdaysGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "stockdays", "update"), m.base.RunAction(m.handler.Update))
		stockdaysGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "stockdays", "delete"), m.base.RunAction(m.handler.Delete))
	}
}
