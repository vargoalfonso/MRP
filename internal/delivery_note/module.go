package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	deliveryNoteHandler "github.com/ganasa18/go-template/internal/delivery_note/handler"
	deliveryNoteService "github.com/ganasa18/go-template/internal/delivery_note/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *deliveryNoteHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       deliveryNoteService.IDeliveryNoteService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *deliveryNoteHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service deliveryNoteService.IDeliveryNoteService,
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

	// 🔐 PRIVATE (JWT)
	deliveryNotePrivate := v1.Group("/delivery-notes")
	deliveryNotePrivate.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		deliveryNotePrivate.POST("/scan", m.base.RunAction(m.handler.ScanDeliveryNoteItem))
		deliveryNotePrivate.POST("/preview-item", m.base.RunAction(m.handler.PreviewItem))

		deliveryNotePrivate.GET("", roleMiddleware.RequirePermission(m.roleService, "delivery_note", "view"), m.base.RunAction(m.handler.GetDeliveryNotes))
		deliveryNotePrivate.POST("", roleMiddleware.RequirePermission(m.roleService, "delivery_note", "create"), m.base.RunAction(m.handler.CreateDeliveryNote))
		deliveryNotePrivate.POST("/preview", roleMiddleware.RequirePermission(m.roleService, "delivery_note", "view"), m.base.RunAction(m.handler.PreviewDN))
		deliveryNotePrivate.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "delivery_note", "view"), m.base.RunAction(m.handler.GetDeliveryNoteByID))
	}
}
