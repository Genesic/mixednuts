package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/Genesic/mixednuts/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
)

type App struct {
	server *grpc.Server
	reg    *prometheus.Registry
	port   int
}

func NewGrpcApp(port int, server *grpc.Server, reg *prometheus.Registry) *App {
	return &App{
		port:   port,
		server: server,
		reg:    reg,
	}
}

func (s *App) Serve(ctx context.Context) error {
	if s.server == nil {
		return errors.New("grpc add initialized without server")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", s.port))
	if err != nil {
		return fmt.Errorf("cat's listen on port %d: %v", s.port, err.Error())
	}

	reflection.Register(s.server)
	logger := logging.FromContext(ctx)
	logger.Infow("server starts", "port", s.port)
	for k, v := range s.server.GetServiceInfo() {
		logger.Infow("service info", k, v)
	}

	// Create HTTP server for prometheus.
	if s.reg != nil {
		httpServer := &http.Server{Handler: promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{}), Addr: fmt.Sprintf("0.0.0.0:%d", 9092)}
		go func() {
			if err := httpServer.ListenAndServe(); err != nil {
				logger.Fatalw("Unable to start a http server for prometheus.")
			}
		}()
	}

	if err = s.server.Serve(lis); err != nil {
		logger.Errorw("failed to start server", "err", err)
		return err
	}

	return nil
}

func (s *App) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return errors.New("server uninitialized")
	}
	logger := logging.FromContext(ctx)
	logger.Infow("start to shutdown server")
	s.server.GracefulStop()
	return nil
}
