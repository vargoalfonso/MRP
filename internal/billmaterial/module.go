// Package billmaterial is the HTTP module for Products > Bill of Material.
package billmaterial

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	bomHandler "github.com/ganasa18/go-template/internal/billmaterial/handler"
	bomService "github.com/ganasa18/go-template/internal/billmaterial/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *bomHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *bomHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	_ bomService.IService, // kept for symmetry with boot.go wiring
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleSvc,
	}
}

// RegisterRoutes — Products > Bill of Material
//
//	GET    /api/v1/products/bom                         list (expandable tree)
//	POST   /api/v1/products/bom                         create wizard
//	GET    /api/v1/products/bom/:id                     detail
//	PUT    /api/v1/products/bom/:id                     update parent BOM header (partial)
//	PUT    /api/v1/products/bom/:id/lines/:line_id      update child node (line + item, partial)
//	DELETE /api/v1/products/bom/:id                     delete parent BOM + all lines
//	DELETE /api/v1/products/bom/:id/children/:child_id  delete selected child subtree only
//	DELETE /api/v1/products/bom/:id/lines/:line_id      delete selected node subtree by line id
//	POST   /api/v1/products/bom/:id/approval            approve or reject (level-based)
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/api/v1/products/bom")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.GET("", roleMiddleware.RequirePermission(m.roleService, "bom", "view"), m.base.RunAction(m.handler.ListBom))
		g.POST("", roleMiddleware.RequirePermission(m.roleService, "bom", "create"), m.base.RunAction(m.handler.CreateBom))
		g.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "bom", "view"), m.base.RunAction(m.handler.GetBomDetail))
		g.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "bom", "update"), m.base.RunAction(m.handler.UpdateBom))
		g.PUT("/:id/lines/:line_id", roleMiddleware.RequirePermission(m.roleService, "bom", "update"), m.base.RunAction(m.handler.UpdateBomChild))
		g.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "bom", "delete"), m.base.RunAction(m.handler.DeleteBom))
		g.DELETE("/:id/children/:child_id", roleMiddleware.RequirePermission(m.roleService, "bom", "delete"), m.base.RunAction(m.handler.DeleteBomChild))
		g.DELETE("/:id/lines/:line_id", roleMiddleware.RequirePermission(m.roleService, "bom", "delete"), m.base.RunAction(m.handler.DeleteBomLine))
		g.POST("/:id/approval", roleMiddleware.RequirePermission(m.roleService, "bom", "approve"), m.base.RunAction(m.handler.ApproveBom))
	}
}
