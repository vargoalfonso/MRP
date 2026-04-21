package outgoingmaterial

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	outHandler "github.com/ganasa18/go-template/internal/outgoing_material/handler"
	outService "github.com/ganasa18/go-template/internal/outgoing_material/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *outHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       outService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *outHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc outService.IService,
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

// RegisterRoutes registers all Outgoing Raw Material endpoints.
//
//	GET    /api/v1/outgoing-raw-materials        list transactions
//	POST   /api/v1/outgoing-raw-materials        create (process) outgoing transaction
//	GET    /api/v1/outgoing-raw-materials/:id    detail
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	g := r.Group("/api/v1/outgoing-raw-materials")
	g.Use(auth)
	g.GET("", perm("outgoing_raw_material", "view"), m.base.RunAction(m.handler.ListOutgoingRM))
	g.POST("", perm("outgoing_raw_material", "create"), m.base.RunAction(m.handler.CreateOutgoingRM))
	g.GET("/:id", perm("outgoing_raw_material", "view"), m.base.RunAction(m.handler.GetOutgoingRMByID))
	g.GET("/form-options", perm("outgoing_raw_material", "view"), m.base.RunAction(m.handler.GetFormOptions))
}
