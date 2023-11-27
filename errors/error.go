package errors

import "google.golang.org/grpc/codes"

type GrpcError interface {
	GetGrpcCode() codes.Code
	GetGrpcMessage() string
}

type HttpError interface {
	ConvertGrpcError() error
}
