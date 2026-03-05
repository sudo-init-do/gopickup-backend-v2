package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	allowOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	if allowOrigins == "" {
		allowOrigins = "*"
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
