// Package billmaterial is the HTTP module for Products > Bill of Material.
package billmaterial

import (
	bomHandler "github.com/ganasa18/go-template/internal/billmaterial/handler"
	bomService "github.com/ganasa18/go-template/internal/billmaterial/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base    *baseHandler.BaseHTTPHandler
	handler *bomHandler.HTTPHandler
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	handler *bomHandler.HTTPHandler,
	_ bomService.IService, // kept for symmetry with boot.go wiring
) appmodule.HTTPModule {
	return &HTTPModule{base: base, handler: handler}
}

// RegisterRoutes — Products > Bill of Material
//
//	GET  /api/v1/products/bom       list (expandable tree)
//	POST /api/v1/products/bom       create wizard
//	GET  /api/v1/products/bom/:id   detail
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/api/v1/products/bom")
	{
		g.GET("", m.base.RunAction(m.handler.ListBom))
		g.POST("", m.base.RunAction(m.handler.CreateBom))
		g.GET("/:id", m.base.RunAction(m.handler.GetBomDetail))
	}
}
