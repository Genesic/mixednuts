package testutils

import (
	"github.com/Genesic/mixednuts/err"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc/status"
)

func VerifyGrpcError(actual error, expect err.GrpcError) {
	e, ok := status.FromError(actual)
	So(ok, ShouldBeTrue)
	So(e.Code(), ShouldEqual, expect.GetGrpcCode())
	So(e.Message(), ShouldEqual, expect.GetGrpcMessage())
}
