// Package upload is the HTTP module for resumable chunked file uploads.
package upload

import (
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	uploadHandler "github.com/ganasa18/go-template/internal/upload/handler"
	uploadService "github.com/ganasa18/go-template/internal/upload/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	base    *baseHandler.BaseHTTPHandler
	handler *uploadHandler.HTTPHandler
}

func NewHTTPModule(
	base *baseHandler.BaseHTTPHandler,
	handler *uploadHandler.HTTPHandler,
	_ uploadService.IService,
) appmodule.HTTPModule {
	return &HTTPModule{base: base, handler: handler}
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
	sessions := g.Group("/sessions")
	{
		sessions.POST("", m.base.RunAction(m.handler.CreateSession))

		session := sessions.Group("/:session_id")
		{
			session.GET("/", m.base.RunAction(m.handler.GetSession))
			session.POST("/chunks/:index", m.base.RunAction(m.handler.UploadChunk))
			session.POST("/complete", m.base.RunAction(m.handler.Complete))
			session.DELETE("/", m.base.RunAction(m.handler.Cancel))
		}
	}
}
