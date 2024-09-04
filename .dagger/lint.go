package main

import "dagger/ccp/internal/dagger"

func (m *CCP) Lint() *Lint {
	return &Lint{
		Source: m.Root,
	}
}

type Lint struct {
	Source *dagger.Directory
}

func (m *Lint) Go() *dagger.Container {
	return dag.GolangciLint(dagger.GolangciLintOpts{
		Version:   "v" + golangciLintVersion,
		GoVersion: goVersion,
	}).
		Run(m.Source, dagger.GolangciLintRunOpts{
			Verbose: true,
		})
}
