package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/artefactual-labs/ccp/internal/cmd/rootcmd"
	"github.com/artefactual-labs/ccp/internal/cmd/servercmd"
	"github.com/artefactual-labs/ccp/internal/version"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func Run(ctx context.Context, out io.Writer, args []string) error {
	rootCommand, rootConfig := rootcmd.New()

	rootCommand.Subcommands = []*ffcli.Command{
		servercmd.New(rootConfig, out),
		version.New(out),
	}

	if err := rootCommand.Parse(args); err != nil {
		return fmt.Errorf("error during Parse: %v", err)
	}

	return rootCommand.Run(ctx)
}
