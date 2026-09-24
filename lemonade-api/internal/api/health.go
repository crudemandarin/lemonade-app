package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterHealth adds GET /api/health, a liveness check for local runs and deploys.
// It does not touch the database: the server only starts after the DB connects.
//
// The path is not /healthz because Cloud Run's front end reserves that exact path
// and answers it with its own 404 before the request reaches the app. Under /api
// it is also reachable through the web proxy, so one public URL checks both services.
func RegisterHealth(router gin.IRouter) {
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
