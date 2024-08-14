package version

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type config struct {
	out io.Writer
}

func New(out io.Writer) *ffcli.Command {
	cfg := config{out}
	fs := flag.NewFlagSet("ccp version", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "ccp version",
		ShortHelp:  "Print version.",
		FlagSet:    fs,
		Exec:       cfg.exec,
	}
}

func (c *config) exec(ctx context.Context, args []string) error {
	fmt.Fprintf(c.out, "CCP version %s (commit %s)\n", version, gitCommit)

	return nil
}
