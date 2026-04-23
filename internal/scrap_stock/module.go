package scrapstock

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	scrapHandler "github.com/ganasa18/go-template/internal/scrap_stock/handler"
	scrapService "github.com/ganasa18/go-template/internal/scrap_stock/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *scrapHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       scrapService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *scrapHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc scrapService.IService,
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

// RegisterRoutes registers all Scrap Stock endpoints.
//
// Scrap prefix (/api/v1/scrap-stocks, /api/v1/scrap-releases):
//
//	GET    /api/v1/scrap-stocks              list scrap stock records
//	POST   /api/v1/scrap-stocks              create scrap stock (manual)
//	GET    /api/v1/scrap-stocks/:id          detail
//	GET    /api/v1/scrap-releases            list release records
//	POST   /api/v1/scrap-releases            create release request (Sell/Dump)
//	GET    /api/v1/scrap-releases/:id        detail
//	PUT    /api/v1/scrap-releases/:id/approve approve or reject a release
//
// Action UI prefix (/api/v1/action-ui):
//
//	GET    /api/v1/action-ui/scrap/incoming  list incoming scrap
//	POST   /api/v1/action-ui/scrap/incoming  incoming scrap scan
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	// --- Scrap Stock Database ---
	ss := r.Group("/api/v1/scrap-stocks")
	ss.Use(auth)
	ss.GET("", perm("scrap", "view"), m.base.RunAction(m.handler.ListScrapStocks))
	ss.POST("", perm("scrap", "create"), m.base.RunAction(m.handler.CreateScrapStock))
	// NOTE: /stats must be registered before /:id
	ss.GET("/stats", perm("scrap", "view"), m.base.RunAction(m.handler.GetStats))
	ss.GET("/:id", perm("scrap", "view"), m.base.RunAction(m.handler.GetScrapStockByID))
	ss.GET("/:id/history-logs", perm("scrap", "view"), m.base.RunAction(m.handler.ListScrapMovements))

	// --- Scrap Release ---
	sr := r.Group("/api/v1/scrap-releases")
	sr.Use(auth)
	sr.GET("", perm("scrap", "view"), m.base.RunAction(m.handler.ListScrapReleases))
	sr.POST("", perm("scrap", "create"), m.base.RunAction(m.handler.CreateScrapRelease))
	sr.GET("/:id", perm("scrap", "view"), m.base.RunAction(m.handler.GetScrapReleaseByID))
	sr.PUT("/:id/approve", perm("scrap", "update"), m.base.RunAction(m.handler.ApproveScrapRelease))

}
