package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
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

// NewGateway wires the gRPC-Gateway with standard health endpoints.
// Accepts optional GatewayOptions to allow customizing routes (e.g. for testing) in a decoupled way.
func NewGateway(
	ctx context.Context,
	grpcAddr string,
	options ...GatewayOption,
) (http.Handler, error) {
	// Create the grpc-gateway multiplexer
	gwMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
	)

	// Register gRPC service handler from the endpoint
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := bookingv1.RegisterBookingServiceHandlerFromEndpoint(ctx, gwMux, grpcAddr, opts)
	if err != nil {
		return nil, err
	}

	// Create standard root HTTP multiplexer
	mux := http.NewServeMux()

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
