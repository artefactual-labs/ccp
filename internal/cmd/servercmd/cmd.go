// nolint: contextcheck
package servercmd

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/peterbourgon/ff/v3/fftoml"
	"go.artefactual.dev/tools/log"

	"github.com/artefactual-labs/ccp/internal/cmd/rootcmd"
	"github.com/artefactual-labs/ccp/internal/version"
)

func New(rootConfig *rootcmd.Config, out io.Writer) *ffcli.Command {
	cfg := Config{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("ccp server", flag.ExitOnError)
	fs.String("config", "", "Configuration file in the TOML file format")
	fs.StringVar(&cfg.sharedDir, "shared-dir", "", "Shared directory")
	fs.StringVar(&cfg.workflow, "workflow", "", "Workflow document")
	fs.StringVar(&cfg.db.driver, "db.driver", "", "Database driver")
	fs.StringVar(&cfg.db.dsn, "db.dsn", "", "Database DSN")
	fs.StringVar(&cfg.api.admin.Addr, "api.admin.addr", ":8000", "Admin API listen address")
	fs.StringVar(&cfg.webui.Addr, "webui.addr", ":8001", "Web UI listen address")
	fs.StringVar(&cfg.gearmin.addr, "gearmin.addr", ":4730", "Gearmin job server listen address")
	fs.StringVar(&cfg.metrics.Addr, "metrics.addr", "", "Prometheus HTTP API listen address")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "server",
		ShortUsage: "ccp server [flags]",
		ShortHelp:  "Start server.",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("CCP"),
			ff.WithEnvVarSplit("_"),
			ff.WithConfigFileFlag("config"),
			ff.WithConfigFileParser(fftoml.Parser),
		},
		Exec: cfg.Exec,
	}
}

func (c *Config) Exec(ctx context.Context, args []string) error {
	logger := log.New(c.out, log.WithDebug(c.rootConfig.Debug), log.WithLevel(c.rootConfig.Verbosity))
	defer log.Sync(logger)

	logger = logger.WithName("server")
	logger.Info("Starting...",
		"version", version.Version(),
		"commit", version.GitCommit(),
		"pid", os.Getpid(),
		"go", runtime.Version(),
	)

	if c.sharedDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			logger.Error(err, "Failed to determine the user configuration directory.")
			return err
		}
		c.sharedDir = filepath.Join(configDir, "ccp", "shared")
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	s := NewServer(logger, c)
	if err := s.Run(); err != nil {
		logger.Error(err, "Failed to start server.")
		s.Close()
		return err
	}

	<-ctx.Done()

	if err := s.Close(); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error(err, "Failed to close server gracefully.")
		}
		return err
	}

	return nil
}
