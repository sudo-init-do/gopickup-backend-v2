package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	// Simple logging to verify requests are reaching the handler
	// log.Println("Health Check HIT")
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
