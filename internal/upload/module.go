// Package upload is the HTTP module for resumable chunked file uploads.
package upload

import (
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	uploadHandler "github.com/ganasa18/go-template/internal/upload/handler"
	uploadService "github.com/ganasa18/go-template/internal/upload/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base          *baseHandler.BaseHTTPHandler
	handler       *uploadHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	handler *uploadHandler.HTTPHandler,
	_ uploadService.IService,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{base: base, handler: handler, authenticator: authenticator, roleService: roleSvc}
}

// RegisterRoutes — /api/v1/uploads
//
//	POST   /sessions                             create session
//	GET    /sessions/:session_id                 get session status (resume)
//	POST   /sessions/:session_id/chunks/:index   upload chunk (raw binary)
//	POST   /sessions/:session_id/complete        assemble & finalise
//	DELETE /sessions/:session_id                 cancel & cleanup
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/api/v1/uploads")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	sessions := g.Group("/sessions")
	{
		// create session — bom:create
		sessions.POST("", roleMiddleware.RequirePermission(m.roleService, "bom", "create"), m.base.RunAction(m.handler.CreateSession))

		session := sessions.Group("/:session_id")
		{
			session.GET("/", m.base.RunAction(m.handler.GetSession))
			// chunk upload & complete — bom:update (mengganti/menambah asset)
			session.POST("/chunks/:index", roleMiddleware.RequirePermission(m.roleService, "bom", "update"), m.base.RunAction(m.handler.UploadChunk))
			session.POST("/complete", roleMiddleware.RequirePermission(m.roleService, "bom", "update"), m.base.RunAction(m.handler.Complete))
			session.DELETE("/", roleMiddleware.RequirePermission(m.roleService, "bom", "update"), m.base.RunAction(m.handler.Cancel))
		}
	}
}
