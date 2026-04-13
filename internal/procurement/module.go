// Package procurement is the HTTP module for the Procurement domain:
// PO Board (Raw Material / Indirect / SubCon), DN list, and the Create PO wizard.
package procurement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	procHandler "github.com/ganasa18/go-template/internal/procurement/handler"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

// HTTPModule wires the procurement handlers into the router.
type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *procHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

// NewHTTPModule creates a new HTTPModule.
func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *procHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleSvc,
	}
}

// RegisterRoutes registers all procurement endpoints.
//
// Base: /api/v1/procurement
//
// PO Board:
//
//	GET  /procurement/purchase-orders/summary      → KPI cards
//	GET  /procurement/purchase-orders              → PO board list (paginated)
//	GET  /procurement/purchase-orders/form-options → wizard dropdown data
//	POST /procurement/purchase-orders/generate     → create PO(s) from budget
//	GET  /procurement/purchase-orders/:po_id       → PO detail + history logs
//
// DN:
//
//	GET  /procurement/incoming-dns                 → DN list (filter by po_number)
//	GET  /procurement/incoming-dns/:dn_id          → DN detail + items
//
// NOTE: static sub-paths (summary, form-options, generate) MUST be registered
// before the wildcard /:po_id to prevent gin from swallowing them.
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)

	g := r.Group("/api/v1/procurement")
	g.Use(auth)

	po := g.Group("/purchase-orders")

	// Static routes first — must come before /:po_id
	po.GET("/summary",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "view"),
		m.base.RunAction(m.handler.GetSummary),
	)
	po.GET("/form-options",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "view"),
		m.base.RunAction(m.handler.GetFormOptions),
	)
	po.POST("/generate",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "create"),
		m.base.RunAction(m.handler.GeneratePO),
	)

	// PO board list
	po.GET("",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "view"),
		m.base.RunAction(m.handler.ListPOBoard),
	)

	// PO detail — wildcard last
	po.GET("/:po_id",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "view"),
		m.base.RunAction(m.handler.GetPODetail),
	)

	// DN list
	g.GET("/incoming-dns",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "view"),
		m.base.RunAction(m.handler.ListDNs),
	)

	// DN detail
	g.GET("/incoming-dns/:dn_id",
		roleMiddleware.RequirePermission(m.roleService, "procurement", "view"),
		m.base.RunAction(m.handler.GetDNDetail),
	)
}
