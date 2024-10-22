package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"unsafe"

	"dagger/ccp/internal/dagger"
)

func (m *CCP) Worker() *Worker {
	return &Worker{}
}

type Worker struct{}

// AnalyzeClientModules generates a report of all Archivematica client modules.
//
// $ dagger call worker analyze-client-modules --dir=https://github.com/artefactual/archivematica#v1.17.0-rc.2:/src/MCPClient/lib/clientScripts
func (m *Worker) AnalyzeClientModules(
	ctx context.Context,
	// +defaultPath="/worker/worker/clientScripts"
	// +ignore=["__init__.py", "__pycache__", "*.pyc", "lib"]
	dir *dagger.Directory,
) *ClientModulesReport {
	return &ClientModulesReport{
		Dir: dir.WithoutFiles([]string{"__init__.py", "__pycache__", "*.pyc", "lib"}),
	}
}

type ClientModulesReport struct {
	Dir *dagger.Directory
}

func (r *ClientModulesReport) Report(ctx context.Context) (*dagger.File, error) {
	return r.generate(ctx)
}

func (r *ClientModulesReport) Review(ctx context.Context) (*dagger.Container, error) {
	report, err := r.generate(ctx)
	if err != nil {
		return nil, err
	}

	ctr := dag.
		Container().
		From("alpine:edge").
		WithExec([]string{"apk", "add", "--no-cache", "uv", "python3"}).
		WithExec([]string{"uv", "tool", "install", "jtbl"}).
		WithFile("report.json", report).
		Terminal(dagger.ContainerTerminalOpts{
			Cmd: []string{"sh", "-c", `cat report.json | /root/.local/bin/jtbl -n | less -S`},
		})

	return ctr, nil
}

func (r *ClientModulesReport) generate(ctx context.Context) (*dagger.File, error) {
	mods := []*ClientModule{}

	entries, err := r.Dir.Entries(ctx)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		report, err := analyzeModule(ctx, r.Dir.File(entry))
		if err != nil {
			return nil, err
		}
		mods = append(mods, report)
	}

	blob, err := json.Marshal(mods)
	if err != nil {
		return nil, err
	}

	f := dag.Directory().WithNewFile("report.json", string(blob)).File("report.json")

	return f, nil
}

type ClientModule struct {
	Name     string
	Size     string
	Callable bool
	contents []byte
}

func analyzeModule(ctx context.Context, file *dagger.File) (*ClientModule, error) {
	report, err := newModReport(ctx, file)
	if err != nil {
		return nil, err
	}

	if bytes.Contains(report.contents, []byte("def call(jobs")) {
		report.Callable = true
	}

	return report, nil
}

func newModReport(ctx context.Context, file *dagger.File) (*ClientModule, error) {
	report := &ClientModule{}

	if name, err := file.Name(ctx); err != nil {
		return nil, err
	} else {
		report.Name = name
	}

	if size, err := file.Size(ctx); err != nil {
		return nil, err
	} else {
		kbs := float64(size) / 1024.0
		report.Size = fmt.Sprintf("%.2f KB", kbs)
	}

	contents, err := file.Contents(ctx)
	if err != nil {
		return report, err
	}
	report.contents = unsafe.Slice(unsafe.StringData(contents), len(contents))

	return report, nil
}
