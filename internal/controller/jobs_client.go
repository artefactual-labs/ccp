package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/store"
	"github.com/artefactual-labs/ccp/internal/workflow"
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
	taskResults, err := l.submitTasks(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}

	exitCode, err := processTaskResults(ctx, l.j, l.config, taskResults)
	if err != nil {
		return uuid.Nil, fmt.Errorf("process task results: %v", err)
	}

	return nextLink(l.j, exitCode)
}

func (l *directoryClientScriptJob) submitTasks(ctx context.Context) ([]*dispatcher.TaskResult, error) {
	rm := l.j.pkg.unit.replacements(l.config.FilterSubdir).update(l.j.chain)
	tasks := []*dispatcher.TaskRequest{prepareTask(rm, l.config)}

	results, err := l.j.dispatcher.Dispatch(ctx, l.j.id, l.config, tasks)
	if err != nil {
		return nil, fmt.Errorf("dispatch: %v", err)
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

	exitCode, err := processTaskResults(ctx, l.j, l.config, taskResults)
	if err != nil {
		return uuid.Nil, fmt.Errorf("process task results: %v", err)
	}

	return nextLink(l.j, exitCode)
}

func (l *filesClientScriptJob) submitTasks(ctx context.Context, filterSubDir string) ([]*dispatcher.TaskResult, error) {
	files, err := l.j.pkg.Files(ctx, l.config.FilterFileEnd, filterSubDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return []*dispatcher.TaskResult{}, nil // Nothing to do.
	}

	rm := l.j.pkg.unit.replacements(filterSubDir).update(l.j.chain)
	tasks := make([]*dispatcher.TaskRequest, 0, len(files))

	for _, fileReplacements := range files {
		rm = rm.with(fileReplacements)
		task := prepareTask(rm, l.config)
		tasks = append(tasks, task)
	}

	results, err := l.j.dispatcher.Dispatch(ctx, l.j.id, l.config, tasks)
	if err != nil {
		return nil, fmt.Errorf("dispatch: %v", err)
	}

	return results, nil
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

	if ret, err := decodeUnitVariableMapWithKey(val, "filterSubDir"); err != nil {
		return "", err
	} else if ret != "" {
		filterSubDir = ret
	}

	return filterSubDir, nil
}

// prepareTask prepares a task request for the dispatcher.
func prepareTask(rm replacementMapping, linkConfig *workflow.LinkStandardTaskConfig) *dispatcher.TaskRequest {
	req := &dispatcher.TaskRequest{
		Arguments:  rm.replaceValues(linkConfig.Arguments),
		StdoutFile: rm.replaceValues(linkConfig.StdoutFile),
		StderrFile: rm.replaceValues(linkConfig.StderrFile),
	}

	if val, ok := rm["fileUUID"]; ok {
		if id, err := uuid.Parse(string(val)); err == nil {
			req.FileID = id
		}
	}

	if val, ok := rm["%relativeLocation%"]; ok {
		if path, err := filepath.Abs(string(val)); err == nil {
			req.RelativeLocation = filepath.Base(path)
		}
	}

	return req
}

// processTasksResults processes a set of task results produced by a client job,
// e.g.: filesClientScriptJob. It returns the highest exist code seen because
// that is the one that will be used to determine the next link in the chain.
func processTaskResults(ctx context.Context, j *job, cfg *workflow.LinkStandardTaskConfig, results []*dispatcher.TaskResult) (int, error) {
	maxExitCode := 0

	for _, item := range results {
		j.metrics.TaskCompleted(
			item.CreatedAt,
			item.FinishedAt,
			cfg.Execute,
			j.wl.Group.String(),
			j.wl.Description.String(),
		)

		// Calculate the maximum exit code.
		if item.ExitCode > maxExitCode {
			maxExitCode = item.ExitCode
		}
	}

	if err := j.updateStatusFromExitCode(ctx, maxExitCode); err != nil {
		return -1, err
	}

	return maxExitCode, nil
}

// nextLink returns the next link in the chain based on the exit code.
func nextLink(job *job, exitCode int) (uuid.UUID, error) {
	if ec, ok := job.wl.ExitCodes[exitCode]; ok {
		if ec.LinkID == nil {
			return uuid.Nil, io.EOF // End of chain.
		}
		return *ec.LinkID, nil
	}

	if job.wl.FallbackLinkID == uuid.Nil {
		return uuid.Nil, io.EOF // End of chain.
	}

	return job.wl.FallbackLinkID, nil
}
