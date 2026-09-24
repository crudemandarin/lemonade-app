package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterHealth adds GET /healthz, a liveness check for local runs and deploys.
// It does not touch the database: the server only starts after the DB connects.
func RegisterHealth(router gin.IRouter) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
