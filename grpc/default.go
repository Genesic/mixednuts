package grpc

import (
	"context"

	"github.com/Genesic/mixednuts/grpc/server_interceptor"
	"github.com/Genesic/mixednuts/logging"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	grpclogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

func DefaultGrpcServer(logger *zap.SugaredLogger) (*grpc.Server, *prometheus.Registry) {
	logTraceID := func(ctx context.Context) grpclogging.Fields {
		requestID, _ := ctx.Value(logging.RequestIDKey).(string)
		return grpclogging.Fields{string(logging.RequestIDKey), requestID}
	}

	// Setup metrics.
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)
	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		requestID, _ := ctx.Value(logging.RequestIDKey).(string)
		return prometheus.Labels{string(logging.RequestIDKey): requestID}
	}

	// Setup metric for panic recoveries.
	panicsTotal := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})
	grpcPanicRecoveryHandler := func(p any) (err error) {
		panicsTotal.Inc()
		logger.Errorw("recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			server_interceptor.RequestIDServerInterceptor(),
			srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
			grpclogging.UnaryServerInterceptor(interceptorLogger(logger), grpclogging.WithFieldsFromContext(logTraceID)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
	)

	srvMetrics.InitializeMetrics(grpcSrv)
	return grpcSrv, reg
}

func interceptorLogger(logger *zap.SugaredLogger) grpclogging.Logger {
	return grpclogging.LoggerFunc(func(_ context.Context, lvl grpclogging.Level, msg string, fields ...any) {
		switch lvl {
		case grpclogging.LevelDebug:
			logger.Debugw(msg, fields...)
		case grpclogging.LevelInfo:
			logger.Infow(msg, fields...)
		case grpclogging.LevelWarn:
			logger.Warnw(msg, fields...)
		case grpclogging.LevelError:
			logger.Errorw(msg, fields...)
		default:
			logger.Error("unknown level", "lvl", lvl)
		}
	})
}
