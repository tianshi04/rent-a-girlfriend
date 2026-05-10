package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// AuthRequired ensures that the request has user identity headers (injected by mesh).
// DEV FALLBACK: If running locally without Mesh (ENABLE_TEST_ROUTES=true) and headers are missing,
// it extracts the claims from the Authorization Bearer token.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-Id")
		userStatus := c.GetHeader("X-User-Status")
		
		// Fallback for local testing without Istio Service Mesh
		if userID == "" && os.Getenv("ENABLE_TEST_ROUTES") == "true" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				parts := strings.Split(token, ".")
				if len(parts) == 3 {
					payload, err := base64.RawURLEncoding.DecodeString(parts[1])
					if err == nil {
						var claims map[string]interface{}
						if json.Unmarshal(payload, &claims) == nil {
							if sub, ok := claims["sub"].(string); ok {
								c.Request.Header.Set("X-User-Id", sub)
								userID = sub
							}
							if role, ok := claims["role"].(string); ok {
								c.Request.Header.Set("X-User-Role", role)
							}
							if status, ok := claims["status"].(string); ok {
								c.Request.Header.Set("X-User-Status", status)
								userStatus = status
							}
						}
					}
				}
			}
		}

		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
			c.Abort()
			return
		}

		// Enforce active account status
		if userStatus != "" && userStatus != string(vo.StatusActive) {
			c.JSON(http.StatusForbidden, gin.H{"error": "account is locked or inactive"})
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
