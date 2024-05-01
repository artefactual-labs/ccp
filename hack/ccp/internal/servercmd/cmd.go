package servercmd

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/peterbourgon/ff/v3/fftoml"
	"go.artefactual.dev/tools/log"

	"github.com/artefactual/archivematica/hack/ccp/internal/rootcmd"
)

const (
	defaultVerbosity int = 0
	minimumVerbosity int = 0
	maximumVerbosity int = 10
)

func (c *Config) RegisterFlags(fs *flag.FlagSet) {
	fs.String("conifg", "", "Configuration file in the TOML file format")
	fs.StringVar(&c.sharedDir, "shared-dir", "", "Shared directory")
	fs.StringVar(&c.workflow, "workflow", "", "Workflow document")
	fs.StringVar(&c.db.driver, "db.driver", "", "Database driver")
	fs.StringVar(&c.db.dsn, "db.dsn", "", "Database DSN")
	fs.StringVar(&c.api.admin.Addr, "api.admin.addr", "", "Admin API listen address")
	fs.StringVar(&c.gearmin.addr, "gearmin.addr", ":4730", "Gearmin job server listen address")

	c.rootConfig.RegisterFlags(fs)
}

func (c *Config) ConfigureFromEnv() {
	if v := os.Getenv("VERBOSITY"); v != "" {
		c.rootConfig.Verbosity = parseVerbosity(v)
	}
}

func parseVerbosity(v string) int {
	value, err := strconv.Atoi(v)
	if err == nil || value < minimumVerbosity || value > maximumVerbosity {
		return defaultVerbosity
	}

	return value
}

func New(rootConfig *rootcmd.Config, out io.Writer) *ffcli.Command {
	cfg := Config{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("ccp server", flag.ExitOnError)
	cfg.RegisterFlags(fs)
	cfg.ConfigureFromEnv()

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
		"version", "TODO", "commit", "TODO",
		"pid", os.Getpid(), "go", runtime.Version(),
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
		logger.Error(err, "Failed to close server gracefully.")
		return err
	}

	return nil
}
