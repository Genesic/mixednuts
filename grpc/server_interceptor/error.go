package server_interceptor

import (
	"context"
	"github.com/Genesic/mixednuts/errors"
	"google.golang.org/grpc"
)

func ErrorHandleInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, oriErr := handler(ctx, req)
		if oriErr != nil {
			if e, ok := oriErr.(errors.Converter); ok {
				return resp, e.ConvertGrpcError()
			}
		}
		return resp, oriErr
	}
}
