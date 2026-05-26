package middleware

import (
	"gopickup/internal/config"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	// Hardcoded trusted origins (Always allowed)
	trustedOrigins := []string{
		"https://app.gopickup.com.ng",
		"https://www.app.gopickup.com.ng",
		"https://main.gopickup.com.ng",
		"https://www.main.gopickup.com.ng",
		"https://gopickup.com.ng",
		"https://www.gopickup.com.ng",
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:8080",
	}

	// Also read from environment variable
	configOrigins := []string{}
	if cfg.CorsOrigins != "" && cfg.CorsOrigins != "*" {
		for _, o := range strings.Split(cfg.CorsOrigins, ",") {
			configOrigins = append(configOrigins, strings.TrimSpace(o))
		}
	}

	// Never allow wildcard origins in production (defense-in-depth; config
	// validation already rejects this, but guard here too).
	allowAll := (cfg.CorsOrigins == "*" || cfg.CorsOrigins == "") && cfg.AppEnv != "production"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Determine if this origin is allowed
		allow := allowAll

		if !allow && origin != "" {
			// Check hardcoded trusted origins
			for _, o := range trustedOrigins {
				if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(origin)) {
					allow = true
					break
				}
			}
		}

		if !allow && origin != "" {
			// Check config origins
			for _, o := range configOrigins {
				if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(origin)) {
					allow = true
					break
				}
			}
		}

		// Log blocked CORS requests
		if !allow && origin != "" {
			log.Printf("CORS BLOCKED: Origin=%s Path=%s", origin, c.Request.URL.Path)
		}

		// Set CORS headers for allowed origins
		if allow && origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		} else if allowAll {
			// When set to *, respond to any origin
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Always respond 204 to OPTIONS preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
