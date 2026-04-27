package action_ui

import (
	"github.com/ganasa18/go-template/config"
	actionHandler "github.com/ganasa18/go-template/internal/action_ui/handler"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *actionHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *actionHandler.HTTPHandler,
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

// RegisterRoutes registers Action UI endpoints.
// Base: /api/v1/action-ui
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)

	// 🔐 base group
	g := r.Group("/api/v1/action-ui")
	g.Use(auth)

	// ================================
	// 📦 INCOMING MATERIAL
	// ================================
	incoming := g.Group("/incoming-material")
	// 🔍 Lookup (scan QR → auto fill)
	// GET /api/v1/action-ui/incoming-material/lookup?packing_number=KB-123456&item_uniq_code=UQ-123
	incoming.GET("/lookup", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.LookupByPackingNumber))
	// 📥 Submit scan incoming
	// POST /api/v1/action-ui/incoming-material/scans
	incoming.POST("/scans", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.CreateIncomingScan))

	// ================================
	// 🏭 PRODUCTION
	// ================================
	production := g.Group("/production")
	// 🔍 Scan Context (QR → get WO + process info)
	// GET /api/v1/action-ui/production/scan-context?uniq=UQ-123
	production.GET("/scan-context", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ScanContext))
	// ▶️ Scan In (start production)
	// POST /api/v1/action-ui/production/scan-in
	production.POST("/scan-in", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.ScanIn))
	// ⏹️ Scan Out (finish process)
	// POST /api/v1/action-ui/production/scan-out
	production.POST("/scan-out", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.ScanOut))

	// ================================
	// 🧪 QC
	// ================================
	qc := g.Group("/qc")
	// POST /api/v1/action-ui/qc/submit
	qc.GET("/list", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ListQCTask))

	// ✅ QC Process (round 1 / 2 / 3)
	qcGroup := qc.Group("/process")
	qcGroup.POST("/approve", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.QCApprove))
	qcGroup.POST("/reject", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.QCReject))
}
