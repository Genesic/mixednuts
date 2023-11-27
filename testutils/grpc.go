package testutils

import (
	"context"
	"log"
	"net"

	"github.com/Genesic/mixednuts/errors"
	grpcApp "github.com/Genesic/mixednuts/grpc"
	"github.com/Genesic/mixednuts/logging"
	. "github.com/smartystreets/goconvey/convey"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type TestGrpcServer struct {
	lis    *bufconn.Listener
	Server *grpc.Server
}

func NewTestGrpcServer(ctx context.Context) *TestGrpcServer {
	logger := logging.FromContext(ctx)
	bufSize := 1024 * 1024
	lis := bufconn.Listen(bufSize)
	grpcServer, _ := grpcApp.DefaultGrpcServer(logger)

	return &TestGrpcServer{
		lis:    lis,
		Server: grpcServer,
	}
}

func (t *TestGrpcServer) Start() {
	if sErr := t.Server.Serve(t.lis); sErr != nil {
		log.Fatalf("Server exited with error: %v", sErr)
	}
}

func (t *TestGrpcServer) Stop() {
	err := t.lis.Close()
	if err != nil {
		log.Printf("error closing listener: %v", err)
	}
	t.Server.Stop()
}

func (t *TestGrpcServer) Dial() *grpc.ClientConn {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return t.lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	So(err, ShouldBeNil)
	return conn
}

func VerifyGrpcError(actual error, expect errors.GrpcError) {
	e, ok := status.FromError(actual)
	So(ok, ShouldBeTrue)
	So(e.Code(), ShouldEqual, expect.GetCode())
	So(e.Message(), ShouldEqual, expect.GetMessage())
}
