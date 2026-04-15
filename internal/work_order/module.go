package work_order

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	woHandler "github.com/ganasa18/go-template/internal/work_order/handler"
	woService "github.com/ganasa18/go-template/internal/work_order/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *woHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       woService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *woHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc woService.IService,
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

// RegisterRoutes registers Work Order endpoints.
// Base: /api/v1/working-order
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	g := r.Group("/api/v1/working-order")
	g.Use(auth)

	wo := g.Group("/work-orders")
	// Static endpoints must be registered before wildcard (/:id/...)
	wo.GET("/form-options/uniq", perm("work_order", "view"), m.base.RunAction(m.handler.UniqFormOptions))
	wo.GET("/form-options/processes", perm("work_order", "view"), m.base.RunAction(m.handler.ProcessFormOptions))
	wo.GET("/summary", perm("work_order", "view"), m.base.RunAction(m.handler.GetWorkOrderSummary))
	wo.GET("", perm("work_order", "view"), m.base.RunAction(m.handler.ListWorkOrders))
	wo.POST("/preview", perm("work_order", "create"), m.base.RunAction(m.handler.PreviewWorkOrder))
	wo.POST("", perm("work_order", "create"), m.base.RunAction(m.handler.CreateWorkOrder))
	wo.POST("/bulk-approval", perm("work_order", "approve"), m.base.RunAction(m.handler.BulkApproval))
	wo.GET("/:id", perm("work_order", "view"), m.base.RunAction(m.handler.GetWorkOrderDetail))
	wo.POST("/:id/approval", perm("work_order", "approve"), m.base.RunAction(m.handler.Approval))
	wo.GET("/:id/qr", perm("work_order", "view"), m.base.RunAction(m.handler.GetWorkOrderQR))

	rm := g.Group("/rm-processing/work-orders")
	rm.GET("/summary", perm("work_order", "view"), m.base.RunAction(m.handler.GetRMProcessingWorkOrderSummary))
	rm.GET("", perm("work_order", "view"), m.base.RunAction(m.handler.ListRMProcessingWorkOrders))
	rm.POST("", perm("work_order", "create"), m.base.RunAction(m.handler.CreateRMProcessingWorkOrder))
	rm.GET("/:id", perm("work_order", "view"), m.base.RunAction(m.handler.GetRMProcessingWorkOrderDetail))
	rm.POST("/:id/approval", perm("work_order", "approve"), m.base.RunAction(m.handler.ApprovalRMProcessing))

	items := g.Group("/work-order-items")
	items.GET("/:id/qr", perm("work_order", "view"), m.base.RunAction(m.handler.GetWorkOrderItemQR))
}
