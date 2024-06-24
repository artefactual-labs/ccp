package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/artefactual/archivematica/hack/ccp/internal/python"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

// directoryClientScriptJob.
//
// Manager: linkTaskManagerDirectories.
// Class: DirectoryClientScriptJob(DecisionJob).
type directoryClientScriptJob struct {
	j      *job
	config *workflow.LinkStandardTaskConfig
}

var _ jobRunner = (*directoryClientScriptJob)(nil)

func newDirectoryClientScriptJob(j *job) (*directoryClientScriptJob, error) {
	ret := &directoryClientScriptJob{
		j:      j,
		config: &workflow.LinkStandardTaskConfig{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *directoryClientScriptJob) exec(ctx context.Context) (uuid.UUID, error) {
	taskResult, err := l.submitTasks(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}

	exitCode := l.j.processTaskResults(l.config, taskResult)
	if err := l.j.updateStatusFromExitCode(ctx, exitCode); err != nil {
		return uuid.Nil, err
	}

	if ec, ok := l.j.wl.ExitCodes[exitCode]; ok {
		if ec.LinkID == nil {
			return uuid.Nil, io.EOF // End of chain.
		}
		return *ec.LinkID, nil
	}

	if l.j.wl.FallbackLinkID == uuid.Nil {
		return uuid.Nil, io.EOF // End of chain.
	}

	return l.j.wl.FallbackLinkID, nil
}

func (l *directoryClientScriptJob) submitTasks(ctx context.Context) (*taskResults, error) {
	rm := l.j.pkg.unit.replacements(l.config.FilterSubdir).update(l.j.chain)
	args := rm.replaceValues(l.config.Arguments)
	stdout := rm.replaceValues(l.config.StdoutFile)
	stderr := rm.replaceValues(l.config.StderrFile)

	taskBackend := newTaskBackend(l.j.logger, l.j.metrics, l.j, l.j.pkg.store, l.j.gearman, l.config)
	if err := taskBackend.submit(ctx, rm, args, false, stdout, stderr); err != nil {
		return nil, err
	}

	results, err := taskBackend.wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait: %v", err)
	}

	return results, nil
}

// filesClientScriptJob.
//
// Manager: linkTaskManagerFiles.
// Class: FilesClientScriptJob(DecisionJob).
type filesClientScriptJob struct {
	j      *job
	config *workflow.LinkStandardTaskConfig
}

var _ jobRunner = (*filesClientScriptJob)(nil)

func newFilesClientScriptJob(j *job) (*filesClientScriptJob, error) {
	ret := &filesClientScriptJob{
		j:      j,
		config: &workflow.LinkStandardTaskConfig{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *filesClientScriptJob) exec(ctx context.Context) (uuid.UUID, error) {
	filterSubDir, err := l.filterSubDir(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("look up filterSubDir: %v", err)
	}

	taskResults, err := l.submitTasks(ctx, filterSubDir)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}

	exitCode := l.j.processTaskResults(l.config, taskResults)
	if err := l.j.updateStatusFromExitCode(ctx, exitCode); err != nil {
		return uuid.Nil, err
	}

	if ec, ok := l.j.wl.ExitCodes[exitCode]; ok {
		if ec.LinkID == nil {
			return uuid.Nil, io.EOF // End of chain.
		}
		return *ec.LinkID, nil
	}

	if l.j.wl.FallbackLinkID == uuid.Nil {
		return uuid.Nil, io.EOF // End of chain.
	}

	return l.j.wl.FallbackLinkID, nil
}

func (l *filesClientScriptJob) submitTasks(ctx context.Context, filterSubDir string) (*taskResults, error) {
	rm := l.j.pkg.unit.replacements(filterSubDir).update(l.j.chain)
	taskBackend := newTaskBackend(l.j.logger, l.j.metrics, l.j, l.j.pkg.store, l.j.gearman, l.config)

	files, err := l.j.pkg.Files(ctx, l.config.FilterFileEnd, filterSubDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return &taskResults{}, nil // Nothing to do.
	}

	for _, fileReplacements := range files {
		rm = rm.with(fileReplacements)
		args := rm.replaceValues(l.config.Arguments)
		stdout := rm.replaceValues(l.config.StdoutFile)
		stderr := rm.replaceValues(l.config.StderrFile)

		if err := taskBackend.submit(ctx, rm, args, false, stdout, stderr); err != nil {
			return nil, err
		}
	}

	res, err := taskBackend.wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait: %v", err)
	}

	return res, nil
}

// filterSubDir returns the directory to filter files on. This path is usually
// defined in the workflow but can be overridden per package in a UnitVariable,
// so we need to look that up.
func (l *filesClientScriptJob) filterSubDir(ctx context.Context) (string, error) {
	filterSubDir := l.config.FilterSubdir

	// Check if filterSubDir has been overridden for this Transfer/SIP.
	val, err := l.j.pkg.store.ReadUnitVar(ctx, l.j.pkg.id, l.j.pkg.packageType(), l.config.Execute)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return filterSubDir, nil
		}
		return "", err
	}

	if val == "" {
		return filterSubDir, nil
	}
	if m, err := python.EvalMap(val); err != nil {
		if override, ok := m["filterSubDir"]; ok {
			filterSubDir = override
		}
	}

	return filterSubDir, nil
}

// outputClientScriptJob.
//
// Manager: linkTaskManagerGetMicroserviceGeneratedListInStdOut.
// Class: OutputClientScriptJob(DecisionJob).
type outputClientScriptJob struct {
	j      *job
	config *workflow.LinkStandardTaskConfig
}

var _ jobRunner = (*outputClientScriptJob)(nil)

func newOutputClientScriptJob(j *job) (*outputClientScriptJob, error) {
	ret := &outputClientScriptJob{
		j:      j,
		config: &workflow.LinkStandardTaskConfig{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

// The list of choices as encoded by the client script:
//
//	{
//	  "default": {"description": "asdf", "uri": "asdf"},
//	  "5c732a52-6cdb-4b50-ac2e-ae10361b019a": {"description": "asdf", "uri": "asdf"},
//	}
type outputClientScriptChoice struct {
	Description string `json:"description"`
	URI         string `json:"uri"`
}

func (l *outputClientScriptJob) exec(ctx context.Context) (uuid.UUID, error) {
	taskResults, err := l.submitTasks(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}

	exitCode := l.j.processTaskResults(l.config, taskResults)
	taskResult := taskResults.First()

	output := map[string]outputClientScriptChoice{}
	if err := json.Unmarshal([]byte(taskResult.Stdout), &output); err != nil {
		l.j.logger.Error(err, "Unable to parse output: %s", taskResult.Stdout)
	} else {
		choices := make([]choice, 0, len(output))
		for _, item := range output {
			choices = append(choices, choice{
				label: item.Description,
				value: [2]string{"", item.URI},
			})
		}
		l.j.chain.choices = choices
	}

	if err := l.j.updateStatusFromExitCode(ctx, exitCode); err != nil {
		return uuid.Nil, err
	}

	if ec, ok := l.j.wl.ExitCodes[exitCode]; ok {
		if ec.LinkID == nil {
			return uuid.Nil, io.EOF // End of chain.
		}
		return *ec.LinkID, nil
	}

	if l.j.wl.FallbackLinkID == uuid.Nil {
		return uuid.Nil, io.EOF // End of chain.
	}

	return uuid.Nil, nil
}

func (l *outputClientScriptJob) submitTasks(ctx context.Context) (*taskResults, error) {
	rm := l.j.pkg.unit.replacements(l.config.FilterSubdir).update(l.j.chain)
	args := rm.replaceValues(l.config.Arguments)
	stdout := rm.replaceValues(l.config.StdoutFile)
	stderr := rm.replaceValues(l.config.StderrFile)

	taskBackend := newTaskBackend(l.j.logger, l.j.metrics, l.j, l.j.pkg.store, l.j.gearman, l.config)
	if err := taskBackend.submit(ctx, rm, args, true, stdout, stderr); err != nil {
		return nil, err
	}

	results, err := taskBackend.wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait: %v", err)
	}

	return results, nil
}
