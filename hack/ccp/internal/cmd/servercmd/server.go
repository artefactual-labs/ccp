package servercmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/gohugoio/hugo/watcher"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/artefactual/archivematica/hack/ccp/internal/api/admin"
	"github.com/artefactual/archivematica/hack/ccp/internal/controller"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/webui"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

type Server struct {
	logger logr.Logger
	ctx    context.Context
	cancel context.CancelFunc
	config *Config

	// Metrics server.
	metrics *metricsServer

	// Data store.
	store store.Store

	// Embedded job server compatible with Gearman.
	gearman *gearmin.Server

	// Filesystem watcher.
	watcher *watcher.Batcher

	// Workflow processor.
	controller *controller.Controller

	// Admin API.
	admin *admin.Server

	// Web UI.
	webui *webui.Server
}

func NewServer(logger logr.Logger, config *Config) *Server {
	s := &Server{
		logger: logger,
		config: config,
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	return s
}

func (s *Server) Run() error {
	s.logger.V(1).Info("Loading workflow.")
	var (
		wf  *workflow.Document
		err error
	)
	if path := s.config.workflow; path != "" {
		wf, err = workflow.LoadFromFile(path)
	} else {
		wf, err = workflow.Default()
	}
	if err != nil {
		return fmt.Errorf("error loading workflow: %v", err)
	}

	s.logger.V(1).Info("Creating metrics server.")
	s.metrics = newMetricsServer(s.logger.WithName("metrics"), s.config.metrics, wf)
	if err := s.metrics.Run(); err != nil {
		return fmt.Errorf("error creating metrics server: %v", err)
	}

	s.logger.V(1).Info("Creating database store.")
	s.store, err = store.New(s.logger.WithName("store"), s.config.db.driver, s.config.db.dsn)
	if err != nil {
		return fmt.Errorf("error creating database store: %v", err)
	}

	s.logger.V(1).Info("Cleaning up database.")
	{
		ctx, cancel := context.WithTimeout(s.ctx, time.Second*10)
		defer cancel()

		err = s.store.RemoveTransientData(ctx)
	}
	if err != nil {
		return fmt.Errorf("error cleaning up database: %v", err)
	}

	s.logger.V(1).Info("Creating shared directories.", "path", s.config.sharedDir)
	if err := createSharedDirs(s.config.sharedDir); err != nil {
		return fmt.Errorf("error creating shared directories: %v", err)
	}

	var (
		processingConfigsDir = filepath.Join(s.config.sharedDir, "sharedMicroServiceTasksConfigs/processingMCPConfigs")
		watchedDir           = filepath.Join(s.config.sharedDir, "watchedDirectories")
	)

	s.logger.V(1).Info("Creating built-in processing configurations.", "path", processingConfigsDir)
	if err := workflow.InstallBuiltinConfigs(processingConfigsDir); err != nil {
		return fmt.Errorf("error creating built-in processing configurations: %v", err)
	}

	s.logger.V(1).Info("Creating Gearman job server.")
	ln, err := net.Listen("tcp", s.config.gearmin.addr)
	if err != nil {
		return fmt.Errorf("error creating gearmin listener: %v", err)
	} else {
		s.gearman = gearmin.NewServer(ln)
	}

	s.logger.V(1).Info("Creating ssclient.")
	retryableClient := retryablehttp.NewClient()
	retryableClient.Logger = httpClientLogger{s.logger.WithName("ssclient").V(2)}
	ssclient, err := ssclient.NewClient(retryableClient.StandardClient(), s.store, s.config.ssclient)
	if err != nil {
		return fmt.Errorf("error creating ssclient: %v", err)
	}

	s.logger.V(1).Info("Creating controller.")
	s.controller = controller.New(s.logger.WithName("controller"), s.metrics.metrics, ssclient, s.store, s.gearman, wf, s.config.sharedDir, watchedDir)
	if err := s.controller.Run(); err != nil {
		return fmt.Errorf("error creating controller: %v", err)
	}

	s.logger.V(1).Info("Creating filesystem watchers.", "path", watchedDir)
	if s.watcher, err = watch(s.logger.WithName("watcher"), s.controller, wf, watchedDir); err != nil {
		return fmt.Errorf("error creating filesystem watchers: %v", err)
	}

	s.logger.V(1).Info("Creating admin API.")
	processingConfigForm := workflow.NewProcessingConfigForm(wf, ssclient)
	if s.admin, err = admin.New(s.logger.WithName("api.admin"), s.config.api.admin, s.controller, s.store, wf, processingConfigForm); err != nil {
		return fmt.Errorf("error creating admin API: %v", err)
	}
	if err := s.admin.Run(); err != nil {
		return fmt.Errorf("error running admin API: %v", err)
	}

	s.logger.V(1).Info("Creating web UI.")
	s.webui = webui.New(s.logger.WithName("webui"), s.config.webui, s.admin.Addr())
	if err := s.webui.Run(); err != nil {
		return fmt.Errorf("error creating web UI: %v", err)
	}

	s.logger.V(1).Info("Ready.")

	return nil
}

func (s *Server) Close() error {
	var errs error

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	s.logger.Info("Shutting down...")

	s.cancel() // Cancel the root context.

	if s.store != nil && s.store.Running() {
		errs = errors.Join(errs, s.store.Close())
	}

	if s.controller != nil {
		errs = errors.Join(errs, s.controller.Close())
	}

	if s.watcher != nil {
		s.watcher.Close()
	}

	if s.admin != nil {
		errs = errors.Join(errs, s.admin.Close(ctx))
	}

	if s.webui != nil {
		errs = errors.Join(errs, s.webui.Close(ctx))
	}

	if s.metrics != nil {
		errs = errors.Join(errs, s.metrics.Close(ctx))
	}

	// TODO: gearmin's Stop method can block indefinitely under certain circumstances.
	done := make(chan struct{}, 1)
	go func() {
		s.gearman.Stop()
		done <- struct{}{}
	}()
	select {
	case <-time.After(time.Second / 2):
	case <-done:
	}

	return errs
}

type httpClientLogger struct {
	logr.Logger
}

func (l httpClientLogger) Printf(msg string, keysAndValues ...any) {
	method, path := "", ""
	if len(keysAndValues) >= 2 {
		if v, ok := keysAndValues[0].(string); ok {
			method = v
		}
		if v, ok := keysAndValues[1].(*url.URL); ok {
			path = v.Path
		}
	}
	l.Info("ssclient", "method", method, "path", path, "client", "github.com/hashicorp/go-retryablehttp")
}
