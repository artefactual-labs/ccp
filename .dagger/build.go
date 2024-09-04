package main

import (
	"dagger/ccp/internal/dagger"
)

func (m *CCP) Build() *Build {
	return &Build{
		Root:     m.Root,
		Frontend: m.Frontend,
	}
}

type Build struct {
	// +private
	Root *dagger.Directory

	// +private
	Frontend *dagger.Directory
}

func (m *Build) WorkerImage() *dagger.Container {
	return m.Root.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: "Dockerfile",
		Target:     "worker",
	})
}

func (m *Build) CCPImage() *dagger.Container {
	return m.Root.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: "Dockerfile",
		Target:     "ccp",
	})
}

func (m *Build) MySQLContainer() *dagger.Container {
	return dag.Container().From("mysql:"+mysqlVersion).
		WithExposedPort(3306).
		WithEnvVariable("MYSQL_ROOT_PASSWORD", "12345")
}

func goModule() *dagger.Go {
	return dag.Go(dagger.GoOpts{Version: goVersion}).
		WithModuleCache(dag.CacheVolume("ccp-go-mod")).
		WithBuildCache(dag.CacheVolume("ccp-go-build"))
}

// ---

// BuildFrontend returns the compiled frontend.
func (m *Build) BuildFrontend() *dagger.Directory {
	return dag.Container().
		From(nodeImage).
		WithWorkdir("/src").
		WithFile("package.json", m.Frontend.File("package.json")).
		WithFile("package-lock.json", m.Frontend.File("package-lock.json")).
		WithExec([]string{"npm", "install-clean"}).
		WithDirectory(".", m.Frontend).
		WithExec([]string{"npm", "run", "build", "--outDir", "."}).
		Directory("./dist")
}

// BuildRuncAssets returns the compiled runc assets.
func (m *Build) BuildRuncAssets() *dagger.Directory {
	files := []*dagger.File{}

	// Include the runc binary.
	files = append(files,
		dag.Container().
			From(curlImage).
			WithExec([]string{"curl", "-Ls", "https://github.com/opencontainers/runc/releases/download/v1.1.14/runc.amd64", "-o", "/tmp/runc.amd64"}).
			WithExec([]string{"chmod", "+x", "/tmp/runc.amd64"}).
			File("/tmp/runc.amd64"),
	)

	// Include the compressed rootfs.
	rootfs := dag.Container().
		From(debianImage).
		WithExec([]string{"apt", "update"}).
		WithExec([]string{"apt", "install", "--yes", "zstd"}).
		WithDirectory("/rootfs", m.WorkerImage().Rootfs()).
		WithEnvVariable("ZSTD_CLEVEL", "10").
		WithExec([]string{"tar", "-I", "zstd", "-cf", "/tmp/rootfs.tar.zst", "/rootfs"}).
		WithExec([]string{"bash", "-c", "hash=($(md5sum /etc/passwd)); echo $hash > /tmp/rootfs.tar.zst.md5"})
	files = append(files,
		rootfs.File("/tmp/rootfs.tar.zst"),
		rootfs.File("/tmp/rootfs.tar.zst.md5"),
	)

	return dag.Directory().WithFiles("/", files)
}

func (m *Build) BuildStandaloneBinary() *dagger.File {
	src := m.Root.
		WithDirectory("internal/webui/assets", m.BuildFrontend()).
		WithDirectory("internal/worker/runc/assets", m.BuildRuncAssets())

	binary := dag.Go(dagger.GoOpts{Version: goVersion}).
		WithCgoDisabled().
		WithSource(src).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      ".",
			Tags:     []string{"worker_runc"},
			Trimpath: true,
			Ldflags: []string{
				"-X 'github.com/artefactual-labs/ccp/internal/version.version='",
				"-X 'github.com/artefactual-labs/ccp/internal/version.gitCommit='",
			},
		})

	return binary
}
