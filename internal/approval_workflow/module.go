package approval_workflow

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	approvalWorkflowHandler "github.com/ganasa18/go-template/internal/approval_workflow/handler"
	approvalWorkflowService "github.com/ganasa18/go-template/internal/approval_workflow/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *approvalWorkflowHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       approvalWorkflowService.IApprovalWorkflowService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *approvalWorkflowHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service approvalWorkflowService.IApprovalWorkflowService,
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

	approvalWorkflowGroup := v1.Group("/approval-workflows")

	// 🔐 wajib login
	approvalWorkflowGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		approvalWorkflowGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "view"), m.base.RunAction(m.handler.GetApprovalWorkflows))
		approvalWorkflowGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "create"), m.base.RunAction(m.handler.CreateApprovalWorkflow))
		approvalWorkflowGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "view"), m.base.RunAction(m.handler.GetApprovalWorkflowByID))
		approvalWorkflowGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "update"), m.base.RunAction(m.handler.UpdateApprovalWorkflow))
		approvalWorkflowGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "delete"), m.base.RunAction(m.handler.DeleteApprovalWorkflow))

		approvalWorkflowGroup.POST("/:id/approve", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "approve"), m.base.RunAction(m.handler.Approve))
		approvalWorkflowGroup.POST("/:id/reject", roleMiddleware.RequirePermission(m.roleService, "approval_workflow", "approve"), m.base.RunAction(m.handler.Reject))
	}
}
