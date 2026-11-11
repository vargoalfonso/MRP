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
	production.GET("/scan-context-machine", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ScanContextMachine))
	production.POST("/scan-machine", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.ScanMachine))
	production.GET("/scan-out-context", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ScanOutContext))
	// [repacking] list packing/kanban per Raw Material untuk modal Repacking.
	production.GET("/rm-packing-list", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.RMPackingList))
	// [repack-sisa] pindahkan sisa material antar packing list RM.
	production.POST("/rm-repack", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.RMRepack))

	// NEW: list WO (dropdown), detail WO (semua uniq), lookup RM (scan RM)
	production.GET("/wo-list", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.WOList))
	production.GET("/wo-detail", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.WODetail))
	production.GET("/raw-material", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.RawMaterialLookup))

	// ▶️ Scan In (start production)
	// POST /api/v1/action-ui/production/scan-in
	production.POST("/scan-in", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.ScanIn))
	// ⏹️ Scan Out (finish process)
	// POST /api/v1/action-ui/production/scan-out
	production.POST("/scan-out", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.ScanOut))

	production.POST("/wo-complete", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.CompleteProduction))

	// [scanin-draft-db] Draft scan-in (seed) bersama lintas gadget.
	// GET list draft satu WO, PUT upsert 1 draft, DELETE hapus draft (saat Mulai Produksi).
	production.GET("/scanin-draft", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ListScanInDrafts))
	production.PUT("/scanin-draft", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.UpsertScanInDraft))
	production.DELETE("/scanin-draft", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.DeleteScanInDraft))

	// ⏹️ Issue List
	// POST /api/v1/action-ui/production/scan-out
	production.GET("/issue/list", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.IssueList))

	// ================================
	// 🧪 QC
	// ================================
	qc := g.Group("/qc")
	// POST /api/v1/action-ui/qc/submit
	qc.GET("/list", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ListQCTask))
	// GET /api/v1/action-ui/qc/rounds?wo_id=&wo_item_id= - ronde tersubmit dari DB
	qc.GET("/rounds", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.QCRounds))
	// [overflow-topup] GET rencana penempatan kelebihan (modal konfirmasi)
	qc.GET("/overflow-preview", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.QCOverflowPreview))

	// ✅ QC Process (round 1 / 2 / Scan Out Baru Lanjut Round 3/Round Final)
	qcGroup := qc.Group("/process")
	qcGroup.POST("/list", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.IssueList))
	qcGroup.POST("/approve", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.QCApprove))
	qcGroup.POST("/reject", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.QCReject))
	qcGroup.POST("/finish", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.QCFinish))

	// ================================
	// QC RETURN (Product Return)
	// ================================
	// POST /api/v1/action-ui/qc-return/scan
	// POST /api/v1/action-ui/qc-return/submit-to-qc
	// GET  /api/v1/action-ui/qc-return/pending-tasks
	// POST /api/v1/action-ui/qc-return/submit-validation
	qcReturn := g.Group("/qc-return")
	qcReturn.POST("/scan", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.ScanReturn))
	qcReturn.POST("/submit-to-qc", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.SubmitReturnToQC))
	qcReturn.GET("/pending-tasks", roleMiddleware.RequirePermission(m.roleService, "action_ui", "view"), m.base.RunAction(m.handler.PendingReturnTasks))
	qcReturn.POST("/submit-validation", roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"), m.base.RunAction(m.handler.SubmitReturnValidation))
}
