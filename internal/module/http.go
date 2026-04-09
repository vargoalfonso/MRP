package module

import "github.com/gin-gonic/gin"

// HTTPModule is the contract implemented by each domain module that exposes
// HTTP endpoints inside the modular monolith.
type HTTPModule interface {
	RegisterRoutes(r gin.IRouter)
}
