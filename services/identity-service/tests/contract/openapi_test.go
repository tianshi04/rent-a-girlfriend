package contract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	gateway "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"
)

type mockKeyProvider struct{}

func (m *mockKeyProvider) GetJWKS() (*port.JWKSResponse, error) {
	return &port.JWKSResponse{}, nil
}

func TestContract_RouteConsistency(t *testing.T) {
	// 1. Load OpenAPI Spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../../../contracts/openapi/identity-service.yaml")
	require.NoError(t, err)
	err = doc.Validate(loader.Context)
	require.NoError(t, err)

	// 2. Initialize Gateway with dummy handlers (we only care about the routing)
	getJWKSHandler := query.NewGetJWKSHandler(&mockKeyProvider{})
	gwHandler, err := gateway.NewGateway(context.Background(), "localhost:50051", getJWKSHandler)
	require.NoError(t, err)

	// 3. Verify all OpenAPI paths exist in the Gateway
	for path, pathItem := range doc.Paths.Map() {
		for method := range pathItem.Operations() {
			// Replace {param} with dummy "123" for path matching
			testPath := path
			for {
				start := strings.Index(testPath, "{")
				if start == -1 {
					break
				}
				end := strings.Index(testPath, "}")
				if end == -1 {
					break
				}
				testPath = testPath[:start] + "123" + testPath[end+1:]
			}

			testFullPath := "/api/v1" + testPath
			if path == "/health" || path == "/.well-known/jwks.json" {
				testFullPath = testPath
			}

			req := httptest.NewRequest(method, testFullPath, nil)
			w := httptest.NewRecorder()
			gwHandler.ServeHTTP(w, req)

			// If the route exists, the gateway will match it and attempt to route it.
			// It may fail (e.g. returning 503 Service Unavailable, 500, or other errors),
			// but it must NOT return 404 Not Found.
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"Route %s %s defined in OpenAPI but missing or not routed in grpc-gateway", method, testFullPath)
		}
	}
}

func TestContract_ResponseSchemaValidation(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../../../contracts/openapi/identity-service.yaml")
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
			path:           "/api/v1/admin/accounts/550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: 200,
			body:           `{"id":"550e8400-e29b-41d4-a716-446655440000","email":"test@example.com","role":"ACCOUNT_ROLE_CLIENT","status":"ACCOUNT_STATUS_ACTIVE","violationCount":0,"createdAt":"2026-05-10T11:00:00Z"}`,
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
