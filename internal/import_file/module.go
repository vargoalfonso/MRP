package import_file

import (
	"github.com/ganasa18/go-template/config"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	importHandler "github.com/ganasa18/go-template/internal/import_file/handler"
	importService "github.com/ganasa18/go-template/internal/import_file/service"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *importHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       importService.ImportService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *importHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service importService.ImportService,
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

	v1.GET("/template/prls", m.base.RunAction(m.handler.DownloadTemplate))

	importGroup := v1.Group("/import/prls")
	// importGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		importGroup.POST("",
			m.base.RunAction(m.handler.BulkImportPRL),
		)

		importGroup.GET("/failed/:filename",
			m.base.RunAction(m.handler.DownloadFailedFile),
		)
	}
}
