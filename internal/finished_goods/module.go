package finishedgoods

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	fgHandler "github.com/ganasa18/go-template/internal/finished_goods/handler"
	fgService "github.com/ganasa18/go-template/internal/finished_goods/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *fgHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       fgService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *fgHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc fgService.IService,
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

// RegisterRoutes registers all Finished Goods endpoints.
//
//	GET    /api/v1/finished-goods                   list FG inventory
//	POST   /api/v1/finished-goods                   create FG record (manual)
//	GET    /api/v1/finished-goods/parameterized-summary dynamic per-row summary by uniq_code
//	GET    /api/v1/finished-goods/form-options/uniq  uniq autocomplete for create form
//	GET    /api/v1/finished-goods/summary            4 dashboard cards
//	GET    /api/v1/finished-goods/status-monitoring  status monitoring + alerts tab
//	GET    /api/v1/finished-goods/:id                detail
//	PUT    /api/v1/finished-goods/:id                update (stock, warehouse, wo)
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	fg := r.Group("/api/v1/finished-goods")
	fg.Use(auth)

	// NOTE: named sub-paths must be registered before /:id to avoid routing conflict
	fg.GET("/form-options/uniq", perm("finished_goods", "view"), m.base.RunAction(m.handler.CreateFormUniqOptions))
	fg.GET("/parameterized-summary", perm("finished_goods", "view"), m.base.RunAction(m.handler.GetParameterizedSummary))
	fg.GET("/summary", perm("finished_goods", "view"), m.base.RunAction(m.handler.GetSummary))
	fg.GET("/status-monitoring", perm("finished_goods", "view"), m.base.RunAction(m.handler.GetStatusMonitoring))

	fg.GET("", perm("finished_goods", "view"), m.base.RunAction(m.handler.ListFinishedGoods))
	fg.POST("", perm("finished_goods", "create"), m.base.RunAction(m.handler.CreateFinishedGoods))
	fg.GET("/:id", perm("finished_goods", "view"), m.base.RunAction(m.handler.GetFinishedGoodsByID))
	fg.PUT("/:id", perm("finished_goods", "update"), m.base.RunAction(m.handler.UpdateFinishedGoods))
}
