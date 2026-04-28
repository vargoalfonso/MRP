package employee

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	employeeHandler "github.com/ganasa18/go-template/internal/employee/handler"
	employeeService "github.com/ganasa18/go-template/internal/employee/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *employeeHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       employeeService.IEmployeeService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *employeeHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service employeeService.IEmployeeService,
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

	employeeGroup := v1.Group("/employee")

	// 🔐 wajib login
	employeeGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		employeeGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "employee", "view"), m.base.RunAction(m.handler.GetEmployees))
		employeeGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "employee", "create"), m.base.RunAction(m.handler.CreateEmployee))
		employeeGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "employee", "view"), m.base.RunAction(m.handler.GetEmployeeByID))
		employeeGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "employee", "update"), m.base.RunAction(m.handler.UpdateEmployee))
		employeeGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "employee", "delete"), m.base.RunAction(m.handler.DeleteEmployee))
	}
}
