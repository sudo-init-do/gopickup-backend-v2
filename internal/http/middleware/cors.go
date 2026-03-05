package middleware

import (
	"gopickup/internal/config"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowOrigins := cfg.CorsOrigins
	if allowOrigins == "" {
		allowOrigins = "*"
	}
	
	// Production Hard Lock: Ensure CORS is strict
	if cfg.AppEnv == "production" && allowOrigins == "*" {
		log.Fatal("SECURITY ERROR: CORS_ALLOW_ORIGINS cannot be '*' in production. Please set specific origins.")
	}

	origins := strings.Split(allowOrigins, ",")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allow := false

		// Check if origin is allowed
		if allowOrigins == "*" {
			allow = true
		} else {
			for _, o := range origins {
				if strings.TrimSpace(o) == origin {
					allow = true
					break
				}
			}
		}

		if allow {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
