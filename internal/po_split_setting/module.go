package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	poSplitSettingHandler "github.com/ganasa18/go-template/internal/po_split_setting/handler"
	poSplitSettingService "github.com/ganasa18/go-template/internal/po_split_setting/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *poSplitSettingHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       poSplitSettingService.IPOSplitSettingService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *poSplitSettingHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service poSplitSettingService.IPOSplitSettingService,
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

	poSplitSettingGroup := v1.Group("/po-split-setting")

	// 🔐 wajib login
	poSplitSettingGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		poSplitSettingGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "po_split_setting", "view"), m.base.RunAction(m.handler.GetAll))
		poSplitSettingGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "po_split_setting", "create"), m.base.RunAction(m.handler.Create))
		poSplitSettingGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "po_split_setting", "view"), m.base.RunAction(m.handler.GetByID))
		poSplitSettingGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "po_split_setting", "update"), m.base.RunAction(m.handler.Update))
		poSplitSettingGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "po_split_setting", "delete"), m.base.RunAction(m.handler.Delete))
	}
}
