package main

import (
	"context"
	"fmt"

	"dagger/ccp/internal/dagger"
)

const (
	goVersion           = "1.23.2"
	golangciLintVersion = "v1.61.0"

	gitURL = "https://github.com/artefactual-labs/ccp.git"

	alpineImage = "alpine:3.20.1"
	mysqlImage  = "mysql:8.4.1"
)

type CCP struct {
	// Root source directory.
	//
	// +private
	Root *dagger.Directory

	// Frontend source directory.
	//
	// +private
	Frontend *dagger.Directory
}

func New(
	// Root source directory.
	//
	// +defaultPath="/"
	// +ignore=["**/.git", "**/.venv", "**/node_modules", "hack/submodules/archivematica-sampledata"]
	root *dagger.Directory,

	// Frontend source directory.
	//
	// +defaultPath="/web"
	// +ignore=["node_modules", "test-results", "playwright-report"]
	frontend *dagger.Directory,
) (*CCP, error) {
	return &CCP{
		Root:     root,
		Frontend: frontend,
	}, nil
}

func (c *CCP) Info(ctx context.Context) error {
	fmt.Println("====> Root")
	entries, err := c.Root.Entries(ctx)
	if err != nil {
		return err
	}
	for _, item := range entries {
		fmt.Println(item)
	}

	fmt.Println("====> Frontend")
	entries, err = c.Frontend.Entries(ctx)
	if err != nil {
		return err
	}
	for _, item := range entries {
		fmt.Println(item)
	}

	return nil
}
