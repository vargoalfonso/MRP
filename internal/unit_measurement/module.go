package unit_measurement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	unitMeasurementHandler "github.com/ganasa18/go-template/internal/unit_measurement/handler"
	unitMeasurementService "github.com/ganasa18/go-template/internal/unit_measurement/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *unitMeasurementHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       unitMeasurementService.IUnitMeasurementService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *unitMeasurementHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service unitMeasurementService.IUnitMeasurementService,
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

	unitMeasurementGroup := v1.Group("/unit-measurement")

	// 🔐 wajib login
	unitMeasurementGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		unitMeasurementGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "unit_measurement", "view"), m.base.RunAction(m.handler.GetAll))
		unitMeasurementGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "unit_measurement", "create"), m.base.RunAction(m.handler.Create))
		unitMeasurementGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "unit_measurement", "view"), m.base.RunAction(m.handler.GetByID))
		unitMeasurementGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "unit_measurement", "update"), m.base.RunAction(m.handler.Update))
		unitMeasurementGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "unit_measurement", "delete"), m.base.RunAction(m.handler.Delete))
	}
}
