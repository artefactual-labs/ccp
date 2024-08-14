package main

import (
	"errors"

	"dagger/ccp/internal/dagger"
)

const (
	goVersion           = "1.22.5"
	golangciLintVersion = "v1.59.1"

	gitURL = "https://github.com/artefactual-labs/ccp.git"

	alpineImage = "alpine:3.20.1"
	mysqlImage  = "mysql:8.4.1"
)

type CCP struct {
	// Project source directory
	// This will become useful once pulling from remote becomes available
	//
	// +private
	Source *dagger.Directory
}

func New(
	// Project source directory.
	// +optional
	source *dagger.Directory,

	// Checkout the repository (at the designated ref) and use it as the source
	// directory instead of the local one.
	// +optional
	ref string,
) (*CCP, error) {
	if source == nil && ref != "" {
		opts := dagger.GitOpts{KeepGitDir: true}
		source = dag.Git(gitURL, opts).Ref(ref).Tree()
	}

	if source == nil {
		return nil, errors.New("either source or ref is required")
	}

	return &CCP{
		Source: source,
	}, nil
}
