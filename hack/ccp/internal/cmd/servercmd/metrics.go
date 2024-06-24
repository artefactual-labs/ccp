package servercmd

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"

	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

type metricsServer struct {
	logger  logr.Logger
	config  metrics.Config
	metrics *metrics.Metrics
	server  *http.Server
	ln      net.Listener
}

func newMetricsServer(logger logr.Logger, config metrics.Config, wf *workflow.Document) *metricsServer {
	s := &metricsServer{
		logger:  logger,
		config:  config,
		metrics: metrics.NewMetrics(wf),
	}

	return s
}

func (s *metricsServer) Run() error {
	if s.config.Addr == "" {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", s.metrics.Handler())

	s.server = &http.Server{
		Handler:           mux,
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

func (s *metricsServer) Close(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	return s.server.Shutdown(ctx)
}
