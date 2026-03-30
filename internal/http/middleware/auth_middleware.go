package middleware

import (
	"gopickup/internal/config"
	"gopickup/internal/utils"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""

		// 1. Try Authorization Header (Most common)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Handle "Bearer <token>" or just "<token>"
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				tokenString = strings.TrimSpace(authHeader[7:])
			} else {
				tokenString = strings.TrimSpace(authHeader)
			}
		}

		// 2. Try Query Parameter Fallback (useful for WebSocket debugging or header-stripping proxies)
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			log.Printf("AUTH_DEBUG: No token found in Authorization header or query param. Header: '%s'", authHeader)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (missing token)"})
			c.Abort()
			return
		}

		claims, err := utils.ValidateJWT(tokenString, cfg.JWTSecret)
		if err != nil {
			log.Printf("AUTH_DEBUG: JWT Validation Failed. Error: %v | Token Length: %d", err, len(tokenString))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid or expired token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RoleGuard(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role not found in context"})
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}
