package product_return

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	productReturnHandler "github.com/ganasa18/go-template/internal/product_return/handler"
	productReturnService "github.com/ganasa18/go-template/internal/product_return/service"

	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *productReturnHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       productReturnService.IProductReturn
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *productReturnHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service productReturnService.IProductReturn,
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

	productReturnGroup := v1.Group("/product-return")

	// wajib login
	productReturnGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		productReturnGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "product_return", "view"), m.base.RunAction(m.handler.GetProductReturns))
		productReturnGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "product_return", "create"), m.base.RunAction(m.handler.CreateProductReturn))
		productReturnGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "product_return", "view"), m.base.RunAction(m.handler.GetProductReturnByID))
		productReturnGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "product_return", "update"), m.base.RunAction(m.handler.UpdateProductReturn))
		productReturnGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "product_return", "delete"), m.base.RunAction(m.handler.DeleteProductReturn))
	}
}
