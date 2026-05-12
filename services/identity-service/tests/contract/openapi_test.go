package contract

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	service_http "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"
	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/handler"
)

func TestContract_RouteConsistency(t *testing.T) {
	// 1. Load OpenAPI Spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../api/openapi/openapi.yaml")
	require.NoError(t, err)
	err = doc.Validate(loader.Context)
	require.NoError(t, err)

	// 2. Initialize Router with dummy handlers (we only care about the routes)
	r := service_http.NewRouter(&handler.AuthHandler{}, &handler.AdminHandler{})
	
	// 3. Extract Gin Routes
	ginRoutes := make(map[string]bool)
	for _, route := range r.Routes() {
		// Convert Gin style :param to OpenAPI style {param}
		path := strings.ReplaceAll(route.Path, ":id", "{id}")
		key := fmt.Sprintf("%s %s", route.Method, path)
		ginRoutes[key] = true
	}

	// 4. Verify all OpenAPI paths exist in Gin
	for path, pathItem := range doc.Paths.Map() {
		for method := range pathItem.Operations() {
			// Skip internal test routes if they are not in OpenAPI
			// (openapi.yaml usually doesn't include /test/mock-login)
			
			// Note: openapi.yaml has /api/v1 as base server URL, but paths are relative to it or absolute
			// In our spec, paths like /health are relative to /api/v1 if server is set, 
			// but wait, look at openapi.yaml:
			// servers: [url: /api/v1]
			// paths: [/health, /auth/google/init, ...]
			// So actual URL is /api/v1/health
			
			fullPath := "/api/v1" + path
			if path == "/health" || path == "/.well-known/jwks.json" {
				fullPath = path // These are root level in router.go
			}

			key := fmt.Sprintf("%s %s", method, fullPath)
			assert.True(t, ginRoutes[key], "Route %s defined in OpenAPI but missing in Gin router", key)
		}
	}
}

func TestContract_ResponseSchemaValidation(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../api/openapi/openapi.yaml")
	require.NoError(t, err)

	// Create a router for validation
	router, err := gorillamux.NewRouter(doc)
	require.NoError(t, err)

	// Test cases for schema validation
	// We check if specific responses match their schemas
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Health Check Response",
			method:         "GET",
			path:           "/api/v1/health",
			expectedStatus: 200,
			body:           `{"status":"ok","service":"identity-service"}`,
		},
		{
			name:           "Token Response Schema",
			method:         "POST",
			path:           "/api/v1/auth/refresh",
			expectedStatus: 200,
			body:           `{"accessToken":"at","refreshToken":"rt","expiresIn":3600}`,
		},
		{
			name:           "Account Response Schema",
			method:         "GET",
			path:           "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: 200,
			body:           `{"id":"550e8400-e29b-41d4-a716-446655440000","email":"test@example.com","role":"CLIENT","status":"ACTIVE","violationCount":0,"createdAt":"2026-05-10T11:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dummy request and response
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp := httptest.NewRecorder()
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(tt.expectedStatus)
			resp.WriteString(tt.body)

			// Find route in OpenAPI
			route, pathParams, err := router.FindRoute(req)
			require.NoError(t, err)

			// Validate Response
			validationInput := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{
					Request:    req,
					PathParams: pathParams,
					Route:      route,
				},
				Status: tt.expectedStatus,
				Header: resp.Header(),
			}

			// Read body for validation
			validationInput.SetBodyBytes(resp.Body.Bytes())

			err = openapi3filter.ValidateResponse(context.Background(), validationInput)
			assert.NoError(t, err, "Response does not match OpenAPI schema for %s %s", tt.method, tt.path)
		})
	}
}
