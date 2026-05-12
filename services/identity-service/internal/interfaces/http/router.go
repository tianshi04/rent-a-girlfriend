package http

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/handler"
	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/middleware"
)

// NewRouter creates and configures the Gin router with all routes.
func NewRouter(authHandler *handler.AuthHandler, adminHandler *handler.AdminHandler) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "identity-service"})
	})

	// JWKS endpoint (public, used by Istio Waypoint)
	r.GET("/.well-known/jwks.json", authHandler.GetJWKS)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.GET("/google/init", authHandler.InitGoogleAuth)
			auth.GET("/google/callback", authHandler.LoginGoogle)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		// Authenticated routes
		authenticated := v1.Group("/")
		authenticated.Use(middleware.AuthRequired())
		{
			// Upgrade requests (authenticated user)
			authenticated.POST("/upgrade-requests", authHandler.RequestUpgrade)

			// Admin-only routes
			adminOnly := authenticated.Group("/")
			adminOnly.Use(middleware.AdminRequired())
			{
				// Account info (internal/admin)
				adminOnly.GET("/accounts/:id", adminHandler.GetAccount)

				// Admin routes
				admin := adminOnly.Group("/admin")
				{
					admin.GET("/upgrade-requests", adminHandler.ListUpgradeRequests)
					admin.PUT("/upgrade-requests/:id/approve", adminHandler.ApproveUpgrade)
					admin.PUT("/upgrade-requests/:id/reject", adminHandler.RejectUpgrade)
					admin.PUT("/accounts/:id/lock", adminHandler.LockAccount)
					admin.PUT("/accounts/:id/unlock", adminHandler.UnlockAccount)
				}
			}
		}

		// Test routes (only enabled in test environment)
		if os.Getenv("ENABLE_TEST_ROUTES") == "true" {
			testGrp := v1.Group("/test")
			{
				testGrp.POST("/mock-login", authHandler.MockLogin)
			}
		}
	}

	return r
}
