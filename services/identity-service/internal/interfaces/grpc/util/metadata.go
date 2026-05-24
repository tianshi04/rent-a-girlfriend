package util

import (
	"context"

	"google.golang.org/grpc/metadata"
)

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

