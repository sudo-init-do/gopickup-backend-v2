package middleware

import (
	"gopickup/internal/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Compare case-insensitively so any historic casing drift in the stored
		// role (e.g. "Client" vs "client") doesn't lock a user out.
		userRole := strings.ToLower(strings.TrimSpace(role.(string)))
		for _, allowed := range allowedRoles {
			if userRole == strings.ToLower(strings.TrimSpace(allowed)) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		c.Abort()
	}
}

// Helper for admin only
func AdminMiddleware() gin.HandlerFunc {
	return RoleMiddleware(string(models.RoleAdmin))
}
