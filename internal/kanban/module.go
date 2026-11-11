package kanban

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	kanbanHandler "github.com/ganasa18/go-template/internal/kanban/handler"
	kanbanService "github.com/ganasa18/go-template/internal/kanban/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *kanbanHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       kanbanService.IKanbanParameterService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *kanbanHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service kanbanService.IKanbanParameterService,
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

	kanbanParameterGroup := v1.Group("/kanban")

	// 🔐 wajib login
	kanbanParameterGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		kanbanParameterGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "kanban", "view"), m.base.RunAction(m.handler.GetKanbanParameters))
		kanbanParameterGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "kanban", "create"), m.base.RunAction(m.handler.CreateKanbanParameter))
		kanbanParameterGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "kanban", "view"), m.base.RunAction(m.handler.GetKanbanParameterByID))
		kanbanParameterGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "kanban", "update"), m.base.RunAction(m.handler.UpdateKanbanParameter))
		kanbanParameterGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "kanban", "delete"), m.base.RunAction(m.handler.DeleteKanbanParameter))
	}
}
