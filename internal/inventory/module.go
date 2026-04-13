package inventory

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	invHandler "github.com/ganasa18/go-template/internal/inventory/handler"
	invService "github.com/ganasa18/go-template/internal/inventory/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *invHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       invService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *invHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc invService.IService,
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

// RegisterRoutes registers all inventory endpoints.
// Base: /api/v1/inventory
//
// Raw Material Database:
//
//	GET    /raw-materials                  list + stats cards
//	POST   /raw-materials                  create
//	POST   /raw-materials/bulk             bulk create
//	GET    /raw-materials/incoming         incoming RM scan list (Action UI tab)
//	GET    /raw-materials/:id              detail
//	PUT    /raw-materials/:id              update
//	DELETE /raw-materials/:id              soft delete
//	GET    /raw-materials/:id/history      movement logs
//
// Indirect Raw Material:
//
//	GET    /indirect-materials             list + stats cards
//	POST   /indirect-materials             create
//	POST   /indirect-materials/bulk        bulk create
//	GET    /indirect-materials/incoming    incoming indirect scan list
//	GET    /indirect-materials/:id         detail
//	PUT    /indirect-materials/:id         update
//	DELETE /indirect-materials/:id         soft delete
//	GET    /indirect-materials/:id/history movement logs
//
// Subcon Inventory:
//
//	GET    /subcon-materials                         stock in vendor list
//	POST   /subcon-materials                         create
//	GET    /subcon-materials/received                stock received from vendor
//	GET    /subcon-materials/:id                     detail
//	PUT    /subcon-materials/:id                     update
//	DELETE /subcon-materials/:id                     soft delete
//	GET    /subcon-materials/:id/history             movement logs
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	g := r.Group("/api/v1/inventory")
	g.Use(auth)

	// --- Raw Material Database ---
	rm := g.Group("/raw-materials")
	rm.GET("", perm("inventory", "view"), m.base.RunAction(m.handler.ListRawMaterials))
	rm.POST("", perm("inventory", "create"), m.base.RunAction(m.handler.CreateRawMaterial))
	rm.POST("/bulk", perm("inventory", "create"), m.base.RunAction(m.handler.BulkCreateRawMaterials))
	// NOTE: static sub-routes (/incoming) must be registered before /:id
	rm.GET("/incoming", perm("inventory", "view"), m.base.RunAction(m.handler.ListIncomingRM))
	rm.GET("/:id", perm("inventory", "view"), m.base.RunAction(m.handler.GetRawMaterialByID))
	rm.PUT("/:id", perm("inventory", "update"), m.base.RunAction(m.handler.UpdateRawMaterial))
	rm.DELETE("/:id", perm("inventory", "delete"), m.base.RunAction(m.handler.DeleteRawMaterial))
	rm.GET("/:id/history", perm("inventory", "view"), m.base.RunAction(m.handler.GetRawMaterialHistory))

	// --- Indirect Raw Material ---
	ind := g.Group("/indirect-materials")
	ind.GET("", perm("inventory", "view"), m.base.RunAction(m.handler.ListIndirectMaterials))
	ind.POST("", perm("inventory", "create"), m.base.RunAction(m.handler.CreateIndirectMaterial))
	ind.POST("/bulk", perm("inventory", "create"), m.base.RunAction(m.handler.BulkCreateIndirectMaterials))
	ind.GET("/incoming", perm("inventory", "view"), m.base.RunAction(m.handler.ListIncomingIndirect))
	ind.GET("/:id", perm("inventory", "view"), m.base.RunAction(m.handler.GetIndirectByID))
	ind.PUT("/:id", perm("inventory", "update"), m.base.RunAction(m.handler.UpdateIndirectMaterial))
	ind.DELETE("/:id", perm("inventory", "delete"), m.base.RunAction(m.handler.DeleteIndirectMaterial))
	ind.GET("/:id/history", perm("inventory", "view"), m.base.RunAction(m.handler.GetIndirectHistory))

	// --- Subcon Inventory ---
	sc := g.Group("/subcon-materials")
	sc.GET("", perm("inventory", "view"), m.base.RunAction(m.handler.ListSubconInventory))
	sc.POST("", perm("inventory", "create"), m.base.RunAction(m.handler.CreateSubconInventory))
	// NOTE: /received must be registered before /:id
	sc.GET("/received", perm("inventory", "view"), m.base.RunAction(m.handler.ListSubconReceived))
	sc.GET("/:id", perm("inventory", "view"), m.base.RunAction(m.handler.GetSubconByID))
	sc.PUT("/:id", perm("inventory", "update"), m.base.RunAction(m.handler.UpdateSubconInventory))
	sc.DELETE("/:id", perm("inventory", "delete"), m.base.RunAction(m.handler.DeleteSubconInventory))
	sc.GET("/:id/history", perm("inventory", "view"), m.base.RunAction(m.handler.GetSubconHistory))

	// --- Kanban Summary (async per-row, used by DN list UI) ---
	g.GET("/kanban-summary", perm("inventory", "view"), m.base.RunAction(m.handler.GetKanbanSummary))
}
