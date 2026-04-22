package wip

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	wipHandler "github.com/ganasa18/go-template/internal/wip/handler"
	wipService "github.com/ganasa18/go-template/internal/wip/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *wipHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       wipService.IWIPService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *wipHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service wipService.IWIPService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleService,
		service:       service,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")

	wipGroup := v1.Group("/wip")

	wipGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		// =========================
		// WIP (HEADER + GENERATE)
		// =========================
		wipGroup.GET("",
			roleMiddleware.RequirePermission(m.roleService, "wip", "view"),
			m.base.RunAction(m.handler.GetAll),
		)

		wipGroup.POST("",
			roleMiddleware.RequirePermission(m.roleService, "wip", "create"),
			m.base.RunAction(m.handler.Create),
		)

		wipGroup.GET("/:id",
			roleMiddleware.RequirePermission(m.roleService, "wip", "view"),
			m.base.RunAction(m.handler.GetByID),
		)

		wipGroup.PUT("/:id",
			roleMiddleware.RequirePermission(m.roleService, "wip", "update"),
			m.base.RunAction(m.handler.Update),
		)

		wipGroup.DELETE("/:id",
			roleMiddleware.RequirePermission(m.roleService, "wip", "delete"),
			m.base.RunAction(m.handler.Delete),
		)

		// =========================
		// WIP ITEMS (READ ONLY)
		// =========================
		wipGroup.GET("/items",
			roleMiddleware.RequirePermission(m.roleService, "wip", "view"),
			m.base.RunAction(m.handler.GetItems),
		)

		// =========================
		// SCAN (CORE 🔥)
		// =========================
		wipGroup.POST("/scan",
			roleMiddleware.RequirePermission(m.roleService, "wip", "update"),
			m.base.RunAction(m.handler.Scan),
		)
	}
}
