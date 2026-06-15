package interceptor

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// TracingInterceptor extracts correlationid from metadata and injects it into context.
// If not present, it generates a new UUID.
func TracingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var corrID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("correlationid"); len(values) > 0 && values[0] != "" {
			corrID = values[0]
		}
	}

	if corrID == "" {
		corrID = uuid.New().String()
	}

	ctx = context.WithValue(ctx, vo.CorrelationIDKey, corrID)
	return handler(ctx, req)
}
