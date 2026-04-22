package adminjobs

import (
	"github.com/ganasa18/go-template/config"
	jobHandler "github.com/ganasa18/go-template/internal/admin_jobs/handler"
	jobService "github.com/ganasa18/go-template/internal/admin_jobs/service"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base         *baseHandler.BaseHTTPHandler
	handler      *jobHandler.HTTPHandler
	adminJobUser string
	adminJobPass string
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *jobHandler.HTTPHandler,
	svc jobService.IService,
) appmodule.HTTPModule {
	return &HTTPModule{
		base:         base,
		handler:      handler,
		adminJobUser: cfg.AdminJobUser,
		adminJobPass: cfg.AdminJobPass,
	}
}

// RegisterRoutes registers admin job endpoints.
// Base: /api/v1/admin/jobs
//
//	POST /rebuild-prl-period-summaries   rebuild inventory_demand_periode_summaries
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/api/v1/admin/jobs")
	g.Use(authMiddleware.BasicAuthMiddleware(m.adminJobUser, m.adminJobPass))

	g.POST("/rebuild-prl-period-summaries", m.base.RunAction(m.handler.RebuildPRLPeriodSummaries))
}
