package base

import (
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

// HTTPModule registers shared base endpoints such as health checks.
type HTTPModule struct {
	handler *baseHandler.BaseHTTPHandler
}

// NewHTTPModule constructs the base HTTP module.
func NewHTTPModule(handler *baseHandler.BaseHTTPHandler) appmodule.HTTPModule {
	return &HTTPModule{handler: handler}
}

// RegisterRoutes implements module.HTTPModule.
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	r.GET("/health", m.handler.HealthCheck)
}
