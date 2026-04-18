package adminjobs

import (
	jobHandler "github.com/ganasa18/go-template/internal/admin_jobs/handler"
	jobService "github.com/ganasa18/go-template/internal/admin_jobs/service"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base          *baseHandler.BaseHTTPHandler
	handler       *jobHandler.HTTPHandler
	authenticator authService.Authenticator
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	handler *jobHandler.HTTPHandler,
	svc jobService.IService,
	authenticator authService.Authenticator,
) appmodule.HTTPModule {
	return &HTTPModule{
		base:          base,
		handler:       handler,
		authenticator: authenticator,
	}
}

// RegisterRoutes registers admin job endpoints.
// Base: /api/v1/admin/jobs
//
//	POST /rebuild-prl-period-summaries   rebuild prl_item_period_summaries cache
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)

	g := r.Group("/api/v1/admin/jobs")
	g.Use(auth)

	g.POST("/rebuild-prl-period-summaries", m.base.RunAction(m.handler.RebuildPRLPeriodSummaries))
}
