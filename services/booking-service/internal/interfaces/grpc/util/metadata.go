package util

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// GetUserID retrieves the user-id injected by Istio waypoint from context metadata.
func GetUserID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("user-id")
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetUserRole retrieves the user-role injected by Istio waypoint from context metadata.
func GetUserRole(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("user-role")
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetUserEmail retrieves the user-email injected by Istio waypoint from context metadata.
func GetUserEmail(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("user-email")
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetAllHeaderValues retrieves all header values for a given key from context metadata.
func GetUserStatus(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("user-status")
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetCorrelationID retrieves the correlationid from incoming context metadata.
func GetCorrelationID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("correlationid")
	if len(values) > 0 && values[0] != "" {
		return values[0]
	}
	return ""
}
