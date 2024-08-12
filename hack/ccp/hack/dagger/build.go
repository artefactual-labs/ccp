package main

import "dagger/ccp/internal/dagger"

func (m *CCP) Build() *Build {
	return &Build{
		Source: m.Source,
	}
}

type Build struct {
	// +private
	Source *dagger.Directory
}

func (m *Build) WorkerImage() *dagger.Container {
	return m.Source.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: "hack/Dockerfile",
		Target:     "archivematica-worker",
	})
}

func (m *Build) CCPImage() *dagger.Container {
	return m.Source.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: "hack/Dockerfile",
		Target:     "archivematica-ccp",
	})
}

func (m *Build) MySQLContainer() *dagger.Container {
	return dag.Container().From(mysqlImage).
		WithExposedPort(3306).
		WithEnvVariable("MYSQL_ROOT_PASSWORD", "12345")
}

func goModule() *dagger.Go {
	return dag.Go(dagger.GoOpts{Version: goVersion}).
		WithModuleCache(dag.CacheVolume("ccp-go-mod")).
		WithBuildCache(dag.CacheVolume("ccp-go-build"))
}
