package prl

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	prlHandler "github.com/ganasa18/go-template/internal/prl/handler"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *prlHandler.HTTPHandler
	authenticator authService.Authenticator
}

func NewHTTPModule(cfg *config.Config, base *baseHandler.BaseHTTPHandler, handler *prlHandler.HTTPHandler, authenticator authService.Authenticator) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	authenticated := v1.Group("")
	authenticated.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		uniqBOMGroup := authenticated.Group("/uniq-boms")
		uniqBOMGroup.POST("", m.base.RunAction(m.handler.CreateUniqBOM))
		uniqBOMGroup.GET("", m.base.RunAction(m.handler.ListUniqBOMs))
		uniqBOMGroup.GET("/:id", m.base.RunAction(m.handler.GetUniqBOM))
		uniqBOMGroup.PUT("/:id", m.base.RunAction(m.handler.UpdateUniqBOM))
		uniqBOMGroup.DELETE("/:id", m.base.RunAction(m.handler.DeleteUniqBOM))

		prlGroup := authenticated.Group("/prls")
		prlGroup.POST("", m.base.RunAction(m.handler.CreatePRL))
		prlGroup.POST("/bulk", m.base.RunAction(m.handler.BulkCreatePRLs))
		prlGroup.GET("", m.base.RunAction(m.handler.ListPRLs))
		prlGroup.GET("/:id/detail", m.base.RunAction(m.handler.GetPRLDetail))
		prlGroup.GET("/:id", m.base.RunAction(m.handler.GetPRL))
		prlGroup.PATCH("/:id", m.base.RunAction(m.handler.UpdatePRL))
		prlGroup.DELETE("/:id", m.base.RunAction(m.handler.DeletePRL))
		prlGroup.POST("/actions/approve", m.base.RunAction(m.handler.ApprovePRLs))
		prlGroup.POST("/actions/reject", m.base.RunAction(m.handler.RejectPRLs))
		prlGroup.POST("/import", m.handler.ImportPRLs)
		prlGroup.GET("/export", m.handler.ExportPRLs)

		lookupGroup := prlGroup.Group("/lookups")
		lookupGroup.GET("/customers", m.base.RunAction(m.handler.ListCustomerLookups))
		lookupGroup.GET("/uniq-boms", m.base.RunAction(m.handler.ListUniqBOMs))
		lookupGroup.GET("/forecast-periods", m.base.RunAction(m.handler.ListForecastPeriods))
	}
}
