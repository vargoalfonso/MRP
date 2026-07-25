package automation

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	autoHandler "github.com/ganasa18/go-template/internal/automation/handler"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

// HTTPModule wires the MRP -> Raigine automation integration endpoints.
type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *autoHandler.HTTPHandler
	authenticator authService.Authenticator
}

// NewHTTPModule constructs the automation HTTP module.
func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *autoHandler.HTTPHandler,
	authenticator authService.Authenticator,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
	}
}

// RegisterRoutes mounts the automation endpoints under /api/v1/automation.
//
//	GET  /api/v1/automation/processes            list automation processes
//	GET  /api/v1/automation/jobs                 list automation job history
//	POST /api/v1/automation/processes/:id/run    trigger a process
//	POST /api/v1/automation/processes/:id/stop   stop a running process
//	POST /api/v1/automation/schedules            create a cron schedule
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	g := v1.Group("/automation")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.GET("/processes", m.base.RunAction(m.handler.ListProcesses))
		g.GET("/jobs", m.base.RunAction(m.handler.ListJobs))
		g.POST("/processes/:id/run", m.base.RunAction(m.handler.Run))
		g.POST("/processes/:id/stop", m.base.RunAction(m.handler.Stop))
		g.POST("/schedules", m.base.RunAction(m.handler.CreateSchedule))
	}
}
