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
	
	// Temporarily allowing '*' for diagnostic sync
	if cfg.AppEnv == "production" && allowOrigins == "*" {
		log.Printf("DIAGNOSTIC WARNING: CORS is currently set to '*' in production.")
	}

	// Pre-calculate allowed origins from config
	configOrigins := strings.Split(allowOrigins, ",")

	// Hardcoded trusted origins (Always allowed)
	trustedOrigins := []string{
		"https://main.gopickup.com.ng",
		"https://www.main.gopickup.com.ng",
		"https://gopickup.com.ng",
		"https://www.gopickup.com.ng",
		"http://localhost:3000",
		"http://localhost:5173",
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		origin := c.Request.Header.Get("Origin")
		allow := false

		// 1. Check trusted origins (Hardcoded whitelist)
		for _, o := range trustedOrigins {
			if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(origin)) {
				allow = true
				break
			}
		}

		// 2. Check configuration if not already allowed
		if !allow {
			if allowOrigins == "*" {
				allow = true
			} else {
				for _, o := range configOrigins {
					if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(origin)) {
						allow = true
						break
					}
				}
			}
		}

		// Log blocked CORS requests for debugging (except health checks)
		if !allow && origin != "" && path != "/" && path != "/health" {
			log.Printf("CORS BLOCKED: Origin=%s Path=%s Allowed=%s", origin, path, allowOrigins)
		}

		if allow {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
