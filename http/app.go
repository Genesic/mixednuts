package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/Genesic/mixednuts/logging"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"time"
)

type MuxServer struct {
	cfg    Config
	server *http.Server

	controllers []Controller
	middlewares []mux.MiddlewareFunc

	additionalHandlers map[string]http.Handler

	port int
}

type Controller interface {
	RegisterHandlers(*mux.Router)
}

type Config struct {
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
}

func NewMuxServer(port int, cfg Config) *MuxServer {
	if cfg.WriteTimeout.Seconds() <= 0 {
		cfg.WriteTimeout = 15 * time.Second
	}

	if cfg.ReadTimeout.Seconds() <= 0 {
		cfg.ReadTimeout = 15 * time.Second
	}

	return &MuxServer{
		cfg:  cfg,
		port: port,
	}
}

func (s *MuxServer) WithMiddlewares(fn ...mux.MiddlewareFunc) *MuxServer {
	s.middlewares = append(s.middlewares, fn...)
	return s
}

func (s *MuxServer) WithControllers(handlers ...Controller) *MuxServer {
	s.controllers = append(s.controllers, handlers...)
	return s
}

func (s *MuxServer) WithAdditionalHandlers(pathHandlerPair ...interface{}) *MuxServer {
	if len(pathHandlerPair)%2 != 0 {
		panic("path and handler numbers don't match")
	}
	if s.additionalHandlers == nil {
		s.additionalHandlers = make(map[string]http.Handler)
	}

	for i := 0; i < len(pathHandlerPair); i += 2 {
		pRaw := pathHandlerPair[i]
		path, pOK := pRaw.(string)
		if !pOK {
			panic(fmt.Sprintf("expect argument %d to be string, got: %v", i, pRaw))
		}

		hRaw := pathHandlerPair[i+1]
		handler, hOK := hRaw.(http.Handler)
		if !hOK {
			panic(fmt.Sprintf("expect argument %d to be http.Handler, got: %v", i+1, hRaw))
		}

		s.additionalHandlers[path] = handler
	}

	return s
}

// Serve starts the server.
func (s *MuxServer) Serve(ctx context.Context) error {
	if s.server != nil {
		return errors.New("server initialized and cannot be reused")
	}
	rootRouter := mux.NewRouter()

	// app routes
	// This comes first before other routes because it's used more frequently.
	appRouter := rootRouter.PathPrefix("/").Subrouter()
	appRouter.Use(s.middlewares...)
	for _, handler := range s.controllers {
		handler.RegisterHandlers(appRouter)
	}

	for path, handler := range s.additionalHandlers {
		rootRouter.Path(path).Handler(handler)
	}

	s.server = &http.Server{
		Handler:      rootRouter,
		Addr:         fmt.Sprintf(":%d", s.port),
		WriteTimeout: s.cfg.WriteTimeout,
		ReadTimeout:  s.cfg.ReadTimeout,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}
	logger := logging.FromContext(ctx)

	logger.Infow("server starts", "address", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Errorw("failed to start server",
			"err", err)
		return err
	}
	return nil
}

func (s *MuxServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return errors.New("server uninitialized")
	}
	return s.server.Shutdown(ctx)
}
