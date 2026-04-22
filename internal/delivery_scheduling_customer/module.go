package delivery_scheduling_customer

import (
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/handler"
	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base          *baseHandler.BaseHTTPHandler
	handler       *handler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       service.IService
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	h *handler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	svc service.IService,
) appmodule.HTTPModule {
	return &HTTPModule{
		base:          base,
		handler:       h,
		authenticator: authenticator,
		roleService:   roleService,
		service:       svc,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")

	// ── Delivery Schedules Customer ──────────────────────────────────────────
	schedules := v1.Group("/delivery-schedules")
	schedules.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		schedules.POST("", roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "create"),
			m.base.RunAction(m.handler.CreateSchedule))

		schedules.GET("/summary",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "view"),
			m.base.RunAction(m.handler.GetSchedulesSummary))

		schedules.GET("",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "view"),
			m.base.RunAction(m.handler.GetSchedulesList))

		// approve-all and approve-partial must be registered before /:id
		schedules.POST("/approve-all",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "approve"),
			m.base.RunAction(m.handler.ApproveAll))

		schedules.POST("/approve-partial",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "approve"),
			m.base.RunAction(m.handler.ApprovePartial))

		schedules.GET("/:id",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "view"),
			m.base.RunAction(m.handler.GetScheduleDetail))

		schedules.POST("/:id/approve",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "approve"),
			m.base.RunAction(m.handler.ApproveSchedule))
	}

	// ── Customer Delivery Notes ───────────────────────────────────────────────
	customerDN := v1.Group("/customer-delivery-notes")
	customerDN.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		customerDN.POST("",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "create"),
			m.base.RunAction(m.handler.CreateCustomerDN))

		customerDN.GET("",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "view"),
			m.base.RunAction(m.handler.GetDNList))

		customerDN.GET("/:id",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "view"),
			m.base.RunAction(m.handler.GetDNDetail))

		customerDN.POST("/:id/confirm",
			roleMiddleware.RequirePermission(m.roleService, "delivery_schedule_customer", "approve"),
			m.base.RunAction(m.handler.ConfirmDN))
	}

	// ── Delivery Scan (Action UI - Customer) ──────────────────────────────────
	scan := v1.Group("/customer-delivery")
	scan.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		scan.GET("/lookup", m.base.RunAction(m.handler.LookupDeliveryItem))
		scan.POST("/scans", m.base.RunAction(m.handler.SubmitDeliveryScan))
	}
}
