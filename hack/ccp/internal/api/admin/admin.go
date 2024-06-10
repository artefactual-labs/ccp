package admin

import (
	"context"
	"net"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/bufbuild/protovalidate-go"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/artefactual/archivematica/hack/ccp/internal/api/corsutil"
	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	adminv1connect "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1/adminv1beta1connect"
	"github.com/artefactual/archivematica/hack/ccp/internal/controller"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

// Server implements the Admin API.
type Server struct {
	logger logr.Logger
	config Config
	ctrl   *controller.Controller
	store  store.Store
	wf     *workflow.Document
	form   *workflow.ProcessingConfigForm
	server *http.Server
	ln     net.Listener
	v      *protovalidate.Validator

	// cache provides an in-memory cache with expiration to prevent concurrent
	// clients from overloading the system.
	cache *ttlcache.Cache[adminv1.PackageType, *adminv1.ListPackagesResponse]
	wg    sync.WaitGroup
}

func New(logger logr.Logger, config Config, ctrl *controller.Controller, store store.Store, wf *workflow.Document, form *workflow.ProcessingConfigForm) (*Server, error) {
	srv := &Server{
		logger: logger,
		config: config,
		ctrl:   ctrl,
		store:  store,
		wf:     wf,
		form:   form,
	}

	if v, err := protovalidate.New(); err != nil {
		return nil, err
	} else {
		srv.v = v
	}

	srv.cache = ttlcache.New[adminv1.PackageType, *adminv1.ListPackagesResponse](
		ttlcache.WithTTL[adminv1.PackageType, *adminv1.ListPackagesResponse](1 * time.Second),
	)
	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()
		srv.cache.Start()
	}()

	return srv, nil
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
			corsutil.New(nil).Handler(mux),
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

func (s *Server) Addr() string {
	return s.ln.Addr().String()
}

func (s *Server) CreatePackage(ctx context.Context, req *connect.Request[adminv1.CreatePackageRequest]) (*connect.Response[adminv1.CreatePackageResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	pkg, err := s.ctrl.Submit(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnknown, nil)
	}

	return connect.NewResponse(&adminv1.CreatePackageResponse{
		Id: pkg.ID().String(),
	}), nil
}

func (s *Server) ReadPackage(ctx context.Context, req *connect.Request[adminv1.ReadPackageRequest]) (*connect.Response[adminv1.ReadPackageResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	id := uuid.MustParse(req.Msg.Id)

	t, err := s.store.ReadTransfer(ctx, id)
	if err == store.ErrNotFound {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	if err != nil {
		s.logger.Error(err, "Failed to read transfer.", "id", id)
		return nil, connect.NewError(connect.CodeUnknown, nil)
	}

	resp := &adminv1.ReadPackageResponse{
		Pkg: &adminv1.Package{
			Id:     req.Msg.Id,
			Name:   t.Name,
			Type:   t.Type,
			Status: t.Status,
		},
	}

	if decisions, ok := s.ctrl.PackageDecisions(id); ok {
		resp.Pkg.Status = adminv1.PackageStatus_PACKAGE_STATUS_AWAITING_DECISION
		resp.Decision = decisions
	}

	resp.Job = []*adminv1.Job{}

	return connect.NewResponse(resp), nil
}

// ListPackages ...
func (s *Server) ListPackages(ctx context.Context, req *connect.Request[adminv1.ListPackagesRequest]) (*connect.Response[adminv1.ListPackagesResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if resp := s.cache.Get(req.Msg.Type); resp != nil {
		return connect.NewResponse(resp.Value()), nil
	}

	// ReadPackagesWithCreationTimestamps hides packages by default, i.e.
	// req.Msg.ExcludeHidden not needed at this point.
	pkgs, err := s.store.ReadPackagesWithCreationTimestamps(ctx, req.Msg.Type)
	if err != nil {
		s.logger.Error(err, "Failed to read packages.")
		return nil, connect.NewError(connect.CodeUnknown, nil)
	}

	// TODO: if we have a SIP, we should provide the access_system_id (transser).

	// Populate directory and jobs for each package.
	for _, pkg := range pkgs {
		pkgID, _ := uuid.Parse(pkg.Id)
		if dir, jobs, err := s.listJobs(ctx, pkgID, true); err != nil {
			s.logger.Error(err, "Failed to read jobs.")
			return nil, connect.NewError(connect.CodeUnknown, nil)
		} else {
			pkg.Name = packageName(pkgID, dir)
			pkg.Directory = dir
			pkg.Job = jobs
		}
	}

	return connect.NewResponse(&adminv1.ListPackagesResponse{
		Package: pkgs,
	}), nil
}

func (s *Server) ListDecisions(ctx context.Context, req *connect.Request[adminv1.ListDecisionsRequest]) (*connect.Response[adminv1.ListDecisionsResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	decisions := []*adminv1.Decision{}
	for _, item := range s.ctrl.Decisions() {
		decisions = slices.Concat(decisions, item)
	}

	return connect.NewResponse(&adminv1.ListDecisionsResponse{
		Decision: decisions,
	}), nil
}

func (s *Server) ResolveDecision(ctx context.Context, req *connect.Request[adminv1.ResolveDecisionRequest]) (*connect.Response[adminv1.ResolveDecisionResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	id := uuid.MustParse(req.Msg.Id)
	err := s.ctrl.ResolveDecision(id, int(req.Msg.Choice.Id))
	if err != nil {
		s.logger.Error(err, "Failed to resolve awaiting decision.", "id", id)
		return nil, connect.NewError(connect.CodeUnknown, nil)
	}

	return connect.NewResponse(&adminv1.ResolveDecisionResponse{}), nil
}

func (s *Server) ListProcessingConfigurationFields(ctx context.Context, req *connect.Request[adminv1.ListProcessingConfigurationFieldsRequest]) (*connect.Response[adminv1.ListProcessingConfigurationFieldsResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	fields, err := s.form.Fields(ctx)
	if err != nil {
		s.logger.Error(err, "Failed to compute some processing configuration fields.")
	}

	return connect.NewResponse(&adminv1.ListProcessingConfigurationFieldsResponse{
		Field: fields,
	}), nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	s.cache.Stop()
	s.wg.Wait()

	return nil
}

func (s *Server) listJobs(ctx context.Context, pkgID uuid.UUID, withDecisions bool) (string, []*adminv1.Job, error) {
	jobs, err := s.store.ListJobs(ctx, pkgID)
	if err != nil {
		return "", nil, err
	}

	// The first item in the list is the most recent, i.e. it contains the
	// current directory.
	dir := ""
	if len(jobs) > 0 {
		dir = jobs[0].Directory
	}

	// We're only doing this to include a workflow that is backward-compatible
	// with the Archivematica Dashboard, but it seems inefficient.
	if withDecisions {
		decisions, ok := s.ctrl.PackageDecisions(pkgID)
		if !ok {
			return dir, jobs, nil
		}
		for _, j := range jobs {
			for _, d := range decisions {
				if j.Id == d.JobId {
					j.Decision = d
				}
			}
		}
	}

	return dir, jobs, nil
}

var (
	matchGroup       = "directory"
	newTransferRegex = regexp.MustCompile(`^.*/(?P<directory>.*)/$`)
	transferRegex    = regexp.MustCompile(`^.*/(?P<directory>.*)-[\w]{8}(-[\w]{4}){3}-[\w]{12}[/]{0,1}$`)
)

func packageName(id uuid.UUID, dir string) string {
	if dir == "" {
		return id.String()
	}

	matches := transferRegex.FindStringSubmatch(dir)
	if len(matches) > 1 {
		for i, name := range transferRegex.SubexpNames() {
			if name == matchGroup {
				return matches[i]
			}
		}
	}

	sep := string(os.PathSeparator)
	if !strings.HasSuffix(dir, sep) {
		dir = dir + sep // TODO: use joinPath util.
	}
	matches = newTransferRegex.FindStringSubmatch(dir)
	if len(matches) > 1 {
		for i, name := range newTransferRegex.SubexpNames() {
			if name == matchGroup {
				return matches[i]
			}
		}
	}

	return id.String()
}
