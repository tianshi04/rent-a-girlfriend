package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
)

// customHeaderMatcher maps incoming HTTP headers to gRPC metadata.
// It matches case-insensitively and allows mesh identity headers to pass as-is.
func customHeaderMatcher(key string) (string, bool) {
	lowerKey := strings.ToLower(key)
	switch lowerKey {
	case "user-id", "user-role", "user-status", "user-email":
		return lowerKey, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

// NewGateway wires the gRPC-Gateway with standard health and JWKS endpoints.
// Accepts optional GatewayOptions to allow customizing routes (e.g. for testing) in a decoupled way.
func NewGateway(
	ctx context.Context,
	grpcAddr string,
	getJWKSHandler *query.GetJWKSHandler,
	options ...GatewayOption,
) (http.Handler, error) {
	// Create the grpc-gateway multiplexer
	gwMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
	)

	// Register gRPC service handler from the endpoint
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := identityv1.RegisterIdentityServiceHandlerFromEndpoint(ctx, gwMux, grpcAddr, opts)
	if err != nil {
		return nil, err
	}

	// Create standard root HTTP multiplexer
	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok", "service": "identity-service"}`))
	})

	// JWKS endpoint (used by Istio Waypoint)
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		jwks, err := getJWKSHandler.Handle()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(jwks)
	})

	// Apply any customized gateway options (e.g., custom routes, test routes)
	for _, opt := range options {
		if opt != nil {
			opt(mux)
		}
	}

	// Route everything else to the grpc-gateway
	mux.Handle("/", gwMux)

	return mux, nil
}
