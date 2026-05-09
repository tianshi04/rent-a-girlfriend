package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// AuthRequired ensures that the request has user identity headers (injected by mesh).
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-Id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminRequired ensures that the request is made by an admin.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetHeader("X-User-Role")
		if role != string(vo.RoleAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
