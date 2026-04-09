// Package po_budget is the HTTP module for Sales > PO Budget Management.
// Sub-menus: Raw Material Budget, Subcon, Indirect.
package po_budget

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	poBudgetHandler "github.com/ganasa18/go-template/internal/po_budget/handler"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *poBudgetHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *poBudgetHandler.HTTPHandler,
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

// RegisterRoutes — Sales > PO Budget Management
//
// Budget type path segment:
//
//	raw-material  → budget_type = raw_material
//	subcon        → budget_type = subcon
//	indirect      → budget_type = indirect
//
// Routes:
//
//	GET    /api/v1/sales/:type/budget              list entries (paginated)
//	GET    /api/v1/sales/:type/budget/aggregate    aggregated by Uniq + Period
//	POST   /api/v1/sales/:type/budget              create entry
//	GET    /api/v1/sales/:type/budget/:id          get single entry
//	PUT    /api/v1/sales/:type/budget/:id          update entry (partial)
//	DELETE /api/v1/sales/:type/budget/:id          delete single entry
//	POST   /api/v1/sales/:type/budget/clear        clear entries (bulk)
//	POST   /api/v1/sales/:type/budget/:id/approve  approve / reject entry
//	POST   /api/v1/sales/:type/budget/import       bulk import CSV
//
//	GET    /api/v1/po-budget/po-split-settings         list split settings
//	PUT    /api/v1/po-budget/po-split-settings/:type   update split setting
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)

	// PO Split Settings
	settings := r.Group("/api/v1/po-budget/po-split-settings")
	settings.Use(auth)
	{
		settings.GET("", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.ListSplitSettings))
		settings.PUT("/:type", roleMiddleware.RequirePermission(m.roleService, "po_budget", "update"), m.base.RunAction(m.handler.UpdateSplitSetting))
	}

	// PRL lookup — Step 1 dropdown + Step 2 item list with allocation
	prl := r.Group("/api/v1/po-budget/prl")
	prl.Use(auth)
	{
		prl.GET("", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.ListPRL))
		prl.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.GetPRLDetail))
	}

	// Budget entries — shared routes for all three sub-menus via :type param
	budget := r.Group("/api/v1/po-budget/:type/budget")
	budget.Use(auth)
	{
		budget.GET("", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.ListEntries))
		budget.GET("/summary", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.GetSummary))
		budget.GET("/aggregate", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.ListAggregated))
		budget.POST("", roleMiddleware.RequirePermission(m.roleService, "po_budget", "create"), m.base.RunAction(m.handler.CreateEntry))
		budget.POST("/bulk", roleMiddleware.RequirePermission(m.roleService, "po_budget", "create"), m.base.RunAction(m.handler.BulkCreateFromPRL))
		budget.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "po_budget", "view"), m.base.RunAction(m.handler.GetEntry))
		budget.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "po_budget", "update"), m.base.RunAction(m.handler.UpdateEntry))
		budget.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "po_budget", "delete"), m.base.RunAction(m.handler.DeleteEntry))
		budget.POST("/clear", roleMiddleware.RequirePermission(m.roleService, "po_budget", "delete"), m.base.RunAction(m.handler.ClearEntries))
		// Single endpoint supports both approve/reject via JSON body.
		// We also expose /reject for frontend convenience.
		budget.POST("/:id/approve", roleMiddleware.RequirePermission(m.roleService, "po_budget", "approve"), m.base.RunAction(m.handler.ApproveEntry))
		budget.POST("/:id/reject", roleMiddleware.RequirePermission(m.roleService, "po_budget", "approve"), m.base.RunAction(m.handler.ApproveEntry))
		budget.POST("/import", roleMiddleware.RequirePermission(m.roleService, "po_budget", "create"), m.base.RunAction(m.handler.ImportEntries))
	}
}
