package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// AuthInterceptor checks for user identity and status in metadata propagated by the service mesh.
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// List of public methods that bypass authentication
	publicMethods := map[string]bool{
		"/identity.v1.IdentityService/InitGoogleAuth": true,
		"/identity.v1.IdentityService/LoginGoogle":    true,
		"/identity.v1.IdentityService/RefreshToken":   true,
		"/identity.v1.IdentityService/Logout":         true,
	}

	if publicMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	userIDs := md.Get("user-id")
	if len(userIDs) == 0 || userIDs[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}

	userStatuses := md.Get("user-status")
	if len(userStatuses) > 0 && userStatuses[0] != "" && userStatuses[0] != string(vo.StatusActive) {
		return nil, status.Error(codes.PermissionDenied, "account is locked or inactive")
	}

	return handler(ctx, req)
}

// AdminInterceptor checks if the user has admin role for protected methods.
func AdminInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// List of methods that require admin role
	adminMethods := map[string]bool{
		"/identity.v1.IdentityService/LockAccount":         true,
		"/identity.v1.IdentityService/UnlockAccount":       true,
		"/identity.v1.IdentityService/ApproveUpgrade":      true,
		"/identity.v1.IdentityService/RejectUpgrade":       true,
		"/identity.v1.IdentityService/ListUpgradeRequests": true,
		"/identity.v1.IdentityService/GetAccount":          true,
	}

	if adminMethods[info.FullMethod] {
		md, _ := metadata.FromIncomingContext(ctx)
		roles := md.Get("user-role")
		if len(roles) == 0 || roles[0] != string(vo.RoleAdmin) {
			return nil, status.Error(codes.PermissionDenied, "admin role required")
		}
	}

	return handler(ctx, req)
}

