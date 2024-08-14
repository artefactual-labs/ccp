package webui

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/artefactual-labs/ccp/internal/api/corsutil"
)

type Server struct {
	logger    logr.Logger
	config    Config
	server    *http.Server
	router    *mux.Router
	adminAddr string
	ln        net.Listener
}

func New(logger logr.Logger, config Config, adminAddr string) *Server {
	s := &Server{
		logger:    logger,
		config:    config,
		router:    mux.NewRouter(),
		adminAddr: adminAddr,
	}

	return s
}

func (s *Server) Run() error {
	if err := s.configureRouter(); err != nil {
		return err
	}

	s.server = &http.Server{
		Handler:           s.router,
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

func (s *Server) configureRouter() error {
	s.router.Use(reportPanic(s.logger))
	s.router.Use(handlers.CompressHandler)
	s.router.Use(corsutil.New(s.config.AllowedOrigins).Handler)
	// TODO: auth, csrf, csp, hsts, xframe, nosniff, xss...

	s.router.HandleFunc("/healthz", s.health)

	target, err := url.Parse("http://" + s.adminAddr)
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { proxy.ServeHTTP(w, r) })
	s.router.PathPrefix("/api").Handler(http.StripPrefix("/api", proxyHandler))

	s.router.PathPrefix("/").Handler(spaHandler(assets))

	return nil
}

func (s *Server) json(w http.ResponseWriter, code int, i interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	err := enc.Encode(i)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	s.json(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (s *Server) Close(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
