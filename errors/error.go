package errors

import "google.golang.org/grpc/codes"

type GrpcError interface {
	GetCode() codes.Code
	GetMessage() string
}

type HttpError interface {
	GetCode() int
	GetMessage() string
}

type Converter interface {
	ConvertGrpcError() error
}
