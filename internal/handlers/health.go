package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheckHandler handles the health check endpoint
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}
