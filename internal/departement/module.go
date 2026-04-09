package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	departementHandler "github.com/ganasa18/go-template/internal/departement/handler"
	departementService "github.com/ganasa18/go-template/internal/departement/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *departementHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       departementService.IDepartement
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *departementHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service departementService.IDepartement,
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

	departmentGroup := v1.Group("/department")

	// 🔐 wajib login
	departmentGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		departmentGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "department", "view"), m.base.RunAction(m.handler.GetDepartments))
		departmentGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "department", "create"), m.base.RunAction(m.handler.CreateDepartment))
		departmentGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "department", "view"), m.base.RunAction(m.handler.GetDepartmentByID))
		departmentGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "department", "update"), m.base.RunAction(m.handler.UpdateDepartment))
		departmentGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "department", "delete"), m.base.RunAction(m.handler.DeleteDepartment))
	}
}
