package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor checks for user identity propagated by the Istio waypoint.
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	userIDs := md.Get("user-id")
	if len(userIDs) == 0 || userIDs[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}

	return handler(ctx, req)
}
