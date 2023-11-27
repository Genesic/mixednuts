package server_interceptor

import (
	"context"
	"github.com/Genesic/mixednuts/logging"
	"github.com/Genesic/mixednuts/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func RequestIDServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.Pairs()
		}
		// Set request ID for context.
		requestIDs := md[string(logging.RequestIDKey)]
		if len(requestIDs) >= 1 {
			ctx = context.WithValue(ctx, logging.RequestIDKey, requestIDs[0])
			return handler(ctx, req)
		}

		// Generate request ID and set context if not exists.
		requestID := utils.GenRequestID()
		ctx = context.WithValue(ctx, logging.RequestIDKey, requestID)
		return handler(ctx, req)
	}
}
