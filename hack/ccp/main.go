package main

import (
	"context"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/artefactual/archivematica/hack/ccp/internal/rootcmd"
	"github.com/artefactual/archivematica/hack/ccp/internal/servercmd"
	"github.com/artefactual/archivematica/hack/ccp/internal/version"
)

func main() {
	out := os.Stderr
	rootCommand, rootConfig := rootcmd.New()

	rootCommand.Subcommands = []*ffcli.Command{
		servercmd.New(rootConfig, out),
		version.New(out),
	}

	if err := rootCommand.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error during Parse: %v\n", err)
		os.Exit(1)
	}

	if err := rootCommand.Run(context.Background()); err != nil {
		os.Exit(1)
	}
}
