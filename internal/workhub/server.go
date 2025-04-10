package workhub

import (
	"context"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/go-logr/logr"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/artefactual-labs/ccp/internal/api/corsutil"
	workerv1connect "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1/workerv1beta1connect"
)

type Server struct {
	logger logr.Logger
	config Config
	hub    *Hub
	server *http.Server
	ln     net.Listener
}

func NewServer(logger logr.Logger, config Config, hub *Hub) *Server {
	return &Server{
		logger: logger,
		config: config,
		hub:    hub,
	}
}

func (s *Server) Run() error {
	compress1KB := connect.WithCompressMinBytes(1024)

	validateInterceptor, err := validate.NewInterceptor()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle(workerv1connect.NewWorkerServiceHandler(
		s.hub,
		connect.WithInterceptors(validateInterceptor),
		compress1KB,
	))
	mux.Handle(grpchealth.NewHandler(
		grpchealth.NewStaticChecker(workerv1connect.WorkerServiceName),
		compress1KB,
	))
	mux.Handle(grpcreflect.NewHandlerV1(
		grpcreflect.NewStaticReflector(workerv1connect.WorkerServiceName),
		compress1KB,
	))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(
		grpcreflect.NewStaticReflector(workerv1connect.WorkerServiceName),
		compress1KB,
	))

	handler := mux
	// auth := authenticate(s.logger, s.store)
	// handler := authn.NewMiddleware(auth).Wrap(mux)

	// A server that supports HTTP 1.1 and HTTP/2 over cleartext (h2c).
	s.server = &http.Server{
		Addr: s.config.Addr,
		Handler: h2c.NewHandler(
			corsutil.New(nil).Handler(handler),
			&http2.Server{},
		),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    8 * 1024, // 8KiB
	}

	if s.ln, err = net.Listen("tcp", s.config.Addr); err != nil {
		return err
	}

	go func() {
		s.logger.Info("Listening...", "addr", s.ln.Addr())
		err := s.server.Serve(s.ln)
		if err != nil && err != http.ErrServerClosed {
			s.logger.Error(err, "Failed to start http.Server")
		}
	}()

	return nil
}

func (s *Server) Addr() net.Addr {
	return s.ln.Addr()
}

func (s *Server) Close(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	s.hub.Close()

	return nil
}
