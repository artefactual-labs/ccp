package admin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/artefactual/archivematica/hack/ccp/internal/api/corsutil"
	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	adminv1connect "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1/adminv1beta1connect"
	"github.com/artefactual/archivematica/hack/ccp/internal/controller"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
)

type Server struct {
	logger logr.Logger
	config Config
	ctrl   *controller.Controller
	store  store.Store
	server *http.Server
	ln     net.Listener
}

func New(logger logr.Logger, config Config, ctrl *controller.Controller, store store.Store) *Server {
	return &Server{
		logger: logger,
		config: config,
		ctrl:   ctrl,
		store:  store,
	}
}

var _ adminv1connect.AdminServiceHandler = (*Server)(nil)

func (s *Server) Run() error {
	compress1KB := connect.WithCompressMinBytes(1024)

	mux := http.NewServeMux()
	mux.Handle(adminv1connect.NewAdminServiceHandler(
		s,
		compress1KB,
	))
	mux.Handle(grpchealth.NewHandler(
		grpchealth.NewStaticChecker(adminv1connect.AdminServiceName),
		compress1KB,
	))
	mux.Handle(grpcreflect.NewHandlerV1(
		grpcreflect.NewStaticReflector(adminv1connect.AdminServiceName),
		compress1KB,
	))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(
		grpcreflect.NewStaticReflector(adminv1connect.AdminServiceName),
		compress1KB,
	))

	s.server = &http.Server{
		Addr: s.config.Addr,
		Handler: h2c.NewHandler(
			corsutil.New().Handler(mux),
			&http2.Server{},
		),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    8 * 1024, // 8KiB
	}

	var err error
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

// validateCreatePackageRequets validates the request.
//
// TODO: use https://github.com/bufbuild/protovalidate.
func validateCreatePackageRequest(msg *adminv1.CreatePackageRequest) error {
	if msg.Name == "" {
		return errors.New("name is empty")
	}

	hasPaths := false
	for _, item := range msg.Path {
		if len(item) > 0 {
			hasPaths = true
			break
		}
	}
	if !hasPaths {
		return errors.New("path is empty")
	}

	if msg.Type == adminv1.TransferType_TRANSFER_TYPE_UNSPECIFIED {
		return errors.New("type is unspecified")
	}

	return nil
}

func (s *Server) CreatePackage(ctx context.Context, req *connect.Request[adminv1.CreatePackageRequest]) (*connect.Response[adminv1.CreatePackageResponse], error) {
	if err := validateCreatePackageRequest(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	resp := &adminv1.CreatePackageResponse{}

	if pkg, err := s.ctrl.Submit(ctx, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, err)
	} else {
		s.logger.Info("TODO: return identifier", "pkg", pkg.Name())
		resp.Id = uuid.New().String()
	}

	return connect.NewResponse(resp), nil
}

func (s *Server) ApproveTransfer(ctx context.Context, req *connect.Request[adminv1.ApproveTransferRequest]) (*connect.Response[adminv1.ApproveTransferResponse], error) {
	return connect.NewResponse(&adminv1.ApproveTransferResponse{
		Id: uuid.New().String(),
	}), nil
}

func (s *Server) ListActivePackages(ctx context.Context, req *connect.Request[adminv1.ListActivePackagesRequest]) (*connect.Response[adminv1.ListActivePackagesResponse], error) {
	return connect.NewResponse(&adminv1.ListActivePackagesResponse{
		Value: s.ctrl.Active(),
	}), nil
}

func (s *Server) ListAwaitingDecisions(ctx context.Context, req *connect.Request[adminv1.ListAwaitingDecisionsRequest]) (*connect.Response[adminv1.ListAwaitingDecisionsResponse], error) {
	return connect.NewResponse(&adminv1.ListAwaitingDecisionsResponse{
		Value: s.ctrl.Decisions(),
	}), nil
}

func (s *Server) ResolveAwaitingDecision(ctx context.Context, req *connect.Request[adminv1.ResolveAwaitingDecisionRequest]) (*connect.Response[adminv1.ResolveAwaitingDecisionResponse], error) {
	return connect.NewResponse(&adminv1.ResolveAwaitingDecisionResponse{}), nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}
