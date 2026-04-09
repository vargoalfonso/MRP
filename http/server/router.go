package server

import (
	appmodule "github.com/ganasa18/go-template/internal/module"
	"github.com/gin-gonic/gin"
)

// setupRoutes registers each module on the shared Gin engine.
func setupRoutes(r *gin.Engine, modules []appmodule.HTTPModule) {
	for _, module := range modules {
		if module == nil {
			continue
		}

		module.RegisterRoutes(r)
	}
}
