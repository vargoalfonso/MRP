package stockopname

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	stockHandler "github.com/ganasa18/go-template/internal/stock_opname/handler"
	stockService "github.com/ganasa18/go-template/internal/stock_opname/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *stockHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       stockService.IService
}

func NewHTTPModule(cfg *config.Config, base *baseHandler.BaseHTTPHandler, handler *stockHandler.HTTPHandler, authenticator authService.Authenticator, roleSvc roleService.IRoleService, svc stockService.IService) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator, roleService: roleSvc, service: svc}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}
	g := r.Group("/api/v1/stock-opname-sessions")
	g.Use(auth)
	g.GET("/stats", perm("stock_opname", "view"), m.base.RunAction(m.handler.GetStats))
	g.GET("/form-options/uniq", perm("stock_opname", "view"), m.base.RunAction(m.handler.ListUniqOptions))
	g.GET("/history-logs", perm("stock_opname", "view"), m.base.RunAction(m.handler.GetHistoryLogs))
	g.GET("", perm("stock_opname", "view"), m.base.RunAction(m.handler.ListSessions))
	g.POST("", perm("stock_opname", "create"), m.base.RunAction(m.handler.CreateSession))
	g.GET("/:id/audit-logs", perm("stock_opname", "view"), m.base.RunAction(m.handler.GetAuditLogs))
	g.GET("/:id", perm("stock_opname", "view"), m.base.RunAction(m.handler.GetSessionByID))
	g.PUT("/:id", perm("stock_opname", "update"), m.base.RunAction(m.handler.UpdateSession))
	g.DELETE("/:id", perm("stock_opname", "delete"), m.base.RunAction(m.handler.DeleteSession))
	g.POST("/:id/entries", perm("stock_opname", "update"), m.base.RunAction(m.handler.AddEntry))
	g.POST("/:id/entries/bulk", perm("stock_opname", "update"), m.base.RunAction(m.handler.BulkAddEntries))
	g.PUT("/:id/entries/:entryId", perm("stock_opname", "update"), m.base.RunAction(m.handler.UpdateEntry))
	g.DELETE("/:id/entries/:entryId", perm("stock_opname", "update"), m.base.RunAction(m.handler.DeleteEntry))
	g.POST("/:id/submit", perm("stock_opname", "update"), m.base.RunAction(m.handler.SubmitSession))
	g.PUT("/:id/approve", perm("stock_opname", "approve"), m.base.RunAction(m.handler.ApproveSession))
	g.PUT("/:id/entries/:entryId/approve", perm("stock_opname", "approve"), m.base.RunAction(m.handler.ApproveEntry))
}
