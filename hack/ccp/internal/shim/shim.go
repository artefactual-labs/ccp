package shim

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"

	"github.com/artefactual/archivematica/hack/ccp/internal/shim/gen"
)

type Server struct {
	logger logr.Logger
	config Config
	server *http.Server
	ln     net.Listener
}

var _ gen.StrictServerInterface = (*Server)(nil)

func NewServer(logger logr.Logger, config Config) *Server {
	return &Server{
		logger: logger,
		config: config,
	}
}

func (s *Server) Run() error {
	s.server = &http.Server{
		Handler:           gen.Handler(gen.NewStrictHandler(s, []gen.StrictMiddlewareFunc{})),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
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
	host, port, err := net.SplitHostPort(s.ln.Addr().String())
	if err != nil {
		return ""
	}
	if host == "" || host == "::" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}

func (s *Server) AdministrationFetchLevelsOfDescription(ctx context.Context, request gen.AdministrationFetchLevelsOfDescriptionRequestObject) (gen.AdministrationFetchLevelsOfDescriptionResponseObject, error) {
	return nil, nil
}

func (s *Server) FilesystemListLevelsOfDescription(ctx context.Context, request gen.FilesystemListLevelsOfDescriptionRequestObject) (gen.FilesystemListLevelsOfDescriptionResponseObject, error) {
	return nil, nil
}

func (s *Server) FilesystemReadMetadata(ctx context.Context, request gen.FilesystemReadMetadataRequestObject) (gen.FilesystemReadMetadataResponseObject, error) {
	return nil, nil
}

func (s *Server) FilesystemUpdateMetadata(ctx context.Context, request gen.FilesystemUpdateMetadataRequestObject) (gen.FilesystemUpdateMetadataResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestListCompleted(ctx context.Context, request gen.IngestListCompletedRequestObject) (gen.IngestListCompletedResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestCopyMetadataFiles(ctx context.Context, request gen.IngestCopyMetadataFilesRequestObject) (gen.IngestCopyMetadataFilesResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestDeleteAll(ctx context.Context, request gen.IngestDeleteAllRequestObject) (gen.IngestDeleteAllResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestReingest(ctx context.Context, request gen.IngestReingestRequestObject) (gen.IngestReingestResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestApproveReingest(ctx context.Context, request gen.IngestApproveReingestRequestObject) (gen.IngestApproveReingestResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestRead(ctx context.Context, request gen.IngestReadRequestObject) (gen.IngestReadResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestListWaiting(ctx context.Context, request gen.IngestListWaitingRequestObject) (gen.IngestListWaitingResponseObject, error) {
	return nil, nil
}

func (s *Server) IngestDelete(ctx context.Context, request gen.IngestDeleteRequestObject) (gen.IngestDeleteResponseObject, error) {
	return nil, nil
}

func (s *Server) ProcessingConfigurationList(ctx context.Context, request gen.ProcessingConfigurationListRequestObject) (gen.ProcessingConfigurationListResponseObject, error) {
	return nil, nil
}

func (s *Server) ProcessingConfigurationDelete(ctx context.Context, request gen.ProcessingConfigurationDeleteRequestObject) (gen.ProcessingConfigurationDeleteResponseObject, error) {
	return nil, nil
}

func (s *Server) ProcessingConfigurationRead(ctx context.Context, request gen.ProcessingConfigurationReadRequestObject) (gen.ProcessingConfigurationReadResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferApprove(ctx context.Context, request gen.TransferApproveRequestObject) (gen.TransferApproveResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferListCompleted(ctx context.Context, request gen.TransferListCompletedRequestObject) (gen.TransferListCompletedResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferDeleteAll(ctx context.Context, request gen.TransferDeleteAllRequestObject) (gen.TransferDeleteAllResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferReingest(ctx context.Context, request gen.TransferReingestRequestObject) (gen.TransferReingestResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferStart(ctx context.Context, request gen.TransferStartRequestObject) (gen.TransferStartResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferRead(ctx context.Context, request gen.TransferReadRequestObject) (gen.TransferReadResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferListUnapproved(ctx context.Context, request gen.TransferListUnapprovedRequestObject) (gen.TransferListUnapprovedResponseObject, error) {
	return nil, nil
}

func (s *Server) TransferDelete(ctx context.Context, request gen.TransferDeleteRequestObject) (gen.TransferDeleteResponseObject, error) {
	return nil, nil
}

func (s *Server) JobsList(ctx context.Context, request gen.JobsListRequestObject) (gen.JobsListResponseObject, error) {
	return nil, nil
}

func (s *Server) PackagesCreate(ctx context.Context, request gen.PackagesCreateRequestObject) (gen.PackagesCreateResponseObject, error) {
	return nil, nil
}

func (s *Server) TasksRead(ctx context.Context, request gen.TasksReadRequestObject) (gen.TasksReadResponseObject, error) {
	return nil, nil
}

func (s *Server) ValidateCreate(ctx context.Context, request gen.ValidateCreateRequestObject) (gen.ValidateCreateResponseObject, error) {
	return nil, nil
}

func (s *Server) Close(ctx context.Context) error {
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}
