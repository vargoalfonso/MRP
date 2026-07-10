// Package machinepattern is the HTTP module for Machine Pattern.
package machinepattern

import (
	mpHandler "github.com/ganasa18/go-template/internal/machinepattern/handler"
	mpService "github.com/ganasa18/go-template/internal/machinepattern/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base    *baseHandler.BaseHTTPHandler
	handler *mpHandler.HTTPHandler
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	handler *mpHandler.HTTPHandler,
	_ mpService.IService,
) appmodule.HTTPModule {
	return &HTTPModule{base: base, handler: handler}
}

// RegisterRoutes — Machine Pattern
//
// GET    /api/v1/machine-patterns
// POST   /api/v1/machine-patterns
// POST   /api/v1/machine-patterns/bulk
// GET    /api/v1/machine-patterns/calculate
// GET    /api/v1/machine-patterns/params
// PUT    /api/v1/machine-patterns/params
// GET    /api/v1/machine-patterns/safety-stock
// GET    /api/v1/machine-patterns/:id
// PUT    /api/v1/machine-patterns/:id
// DELETE /api/v1/machine-patterns/:id
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/api/v1/machine-patterns")
	{
		g.GET("", m.base.RunAction(m.handler.List))
		g.POST("", m.base.RunAction(m.handler.Create))
		g.POST("/bulk", m.base.RunAction(m.handler.BulkCreate))
		g.GET("/calculate", m.base.RunAction(m.handler.Calculate))
		g.GET("/params", m.base.RunAction(m.handler.GetParams))
		g.PUT("/params", m.base.RunAction(m.handler.UpdateParams))
		g.GET("/safety-stock", m.base.RunAction(m.handler.SafetyStock))
		g.GET("/:id", m.base.RunAction(m.handler.GetByID))
		g.PUT("/:id", m.base.RunAction(m.handler.Update))
		g.DELETE("/:id", m.base.RunAction(m.handler.Delete))
	}
}
