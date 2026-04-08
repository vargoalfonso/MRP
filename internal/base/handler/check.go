package handler

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns service and database status. No auth required.
func (h *BaseHTTPHandler) HealthCheck(c *gin.Context) {
	info := gin.H{"service": "up", "db": "unknown"}

	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if len(s.Value) >= 7 {
					info["version"] = s.Value[:7]
				}
			case "vcs.time":
				info["built_at"] = s.Value
			}
		}
	}

	sqlDB, err := h.DB.DB()
	if err == nil && sqlDB.Ping() == nil {
		info["db"] = "up"
	} else {
		info["db"] = "down"
	}

	c.JSON(http.StatusOK, info)
}
