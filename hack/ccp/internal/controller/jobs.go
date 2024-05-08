package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/artefactual/archivematica/hack/ccp/internal/python"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/sqlcmysql"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

type job struct {
	logger logr.Logger

	// gearman is used to dispatch jobs to MCPClient.
	gearman *gearmin.Server

	// id of the job.
	id uuid.UUID

	// createdAt is populated when the job is first created.
	createdAt time.Time

	// chain carries state across multiple jobs within a workflow chain.
	chain *chain

	// pkg is the package related to the job execution.
	pkg *Package

	// wl is the base configuration of each job.
	wl *workflow.Link

	// wf is used to validate preconfigured choices.
	wf *workflow.Document

	// jobRunner is what makes a job executable.
	jobRunner
}

type jobRunner interface {
	exec(context.Context) (uuid.UUID, error)
}

func newJob(logger logr.Logger, chain *chain, pkg *Package, gearman *gearmin.Server, wl *workflow.Link, wf *workflow.Document) (*job, error) {
	j := &job{
		gearman:   gearman,
		id:        uuid.New(),
		createdAt: time.Now().UTC(),
		chain:     chain,
		pkg:       pkg,
		wl:        wl,
		wf:        wf,
	}

	var err error
	switch wl.Manager {

	// Decision jobs - handles workflow decision points.
	case "linkTaskManagerGetUserChoiceFromMicroserviceGeneratedList":
		j.logger = logger.WithName("outputDecisionJob")
		j.jobRunner, err = newOutputDecisionJob(j)
	case "linkTaskManagerChoice":
		j.logger = logger.WithName("nextChainDecisionJob")
		j.jobRunner, err = newNextChainDecisionJob(j)
	case "linkTaskManagerReplacementDicFromChoice":
		j.logger = logger.WithName("updateContextDecisionJob")
		j.jobRunner, err = newUpdateContextDecisionJob(j)

	// Executable jobs - dispatched to the worker pool.
	case "linkTaskManagerDirectories":
		j.logger = logger.WithName("directoryClientScriptJob")
		j.jobRunner, err = newDirectoryClientScriptJob(j)
	case "linkTaskManagerFiles":
		j.logger = logger.WithName("filesClientScriptJob")
		j.jobRunner, err = newFilesClientScriptJob(j)
	case "linkTaskManagerGetMicroserviceGeneratedListInStdOut":
		j.logger = logger.WithName("outputClientScriptJob")
		j.jobRunner, err = newOutputClientScriptJob(j)

	// Local jobs - executed directly.
	case "linkTaskManagerSetUnitVariable":
		j.logger = logger.WithName("setUnitVarLinkJob")
		j.jobRunner, err = newSetUnitVarLinkJob(j)
	case "linkTaskManagerUnitVariableLinkPull":
		j.logger = logger.WithName("getUnitVarLinkJob")
		j.jobRunner, err = newGetUnitVarLinkJob(j)

	default:
		err = fmt.Errorf("unknown job manager: %q", wl.Manager)
	}

	return j, err
}

func (j *job) save(ctx context.Context) error {
	return j.pkg.store.CreateJob(ctx, &sqlcmysql.CreateJobParams{
		ID:                j.id,
		Type:              j.wl.Description.String(),
		CreatedAt:         j.createdAt,
		Createdtimedec:    fmt.Sprintf("%.9f", float64(j.createdAt.Nanosecond())/1e9),
		Directory:         j.pkg.PathForDB(),
		SIPID:             j.pkg.id,
		Unittype:          j.pkg.jobUnitType(),
		Currentstep:       3,
		Microservicegroup: j.wl.Group.String(),
		Hidden:            false,
		Microservicechainlinkspk: sql.NullString{
			String: j.wl.ID.String(),
			Valid:  true,
		},
	})
}

// markAwaitingDecision is used by decision jobs to persist the awaiting status.
func (j *job) markAwaitingDecision(ctx context.Context) error {
	err := j.pkg.store.UpdateJobStatus(ctx, j.id, "STATUS_AWAITING_DECISION")
	if err != nil {
		return fmt.Errorf("mark awaiting decision: %v", err)
	}

	return nil
}

// markComplete is used by decision jobs to persist the completion status.
func (j *job) markComplete(ctx context.Context) error {
	err := j.pkg.store.UpdateJobStatus(ctx, j.id, "STATUS_COMPLETED_SUCCESSFULLY")
	if err != nil {
		return fmt.Errorf("mark complete: %v", err)
	}

	return nil
}

func (j *job) updateStatusFromExitCode(ctx context.Context, code int) error {
	status := ""
	if ec, ok := j.wl.ExitCodes[code]; ok {
		status = ec.JobStatus
	} else {
		status = j.wl.FallbackJobStatus
	}

	err := j.pkg.store.UpdateJobStatus(ctx, j.id, status)
	if err != nil {
		return fmt.Errorf("update job status from exit code: %v", err)
	}

	return nil
}

// outputDecisionJob.
//
// A job that handles a workflow decision point, with choices based on script
// output.
//
// Manager: linkTaskManagerGetUserChoiceFromMicroserviceGeneratedList.
// Class: OutputDecisionJob(DecisionJob).
type outputDecisionJob struct {
	j      *job
	config *workflow.LinkStandardTaskConfig
}

var _ jobRunner = (*outputDecisionJob)(nil)

func newOutputDecisionJob(j *job) (*outputDecisionJob, error) {
	config, ok := j.wl.Config.(workflow.LinkStandardTaskConfig)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &outputDecisionJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *outputDecisionJob) exec(ctx context.Context) (uuid.UUID, error) {
	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	panic("not implemented")

	return uuid.Nil, nil // nolint: govet
}

// nextChainDecisionJob.
//
// A type of workflow decision that determines the next chain to be executed,
// by UUID.
//
// Manager: linkTaskManagerChoice.
// Class: NextChainDecisionJob(DecisionJob).
type nextChainDecisionJob struct {
	j      *job
	config *workflow.LinkMicroServiceChainChoice
}

var _ jobRunner = (*nextChainDecisionJob)(nil)

func newNextChainDecisionJob(j *job) (*nextChainDecisionJob, error) {
	config, ok := j.wl.Config.(workflow.LinkMicroServiceChainChoice)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &nextChainDecisionJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *nextChainDecisionJob) exec(ctx context.Context) (_ uuid.UUID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("nextChainDecisionJob: %v", err)
			return
		}
		if e := l.j.markComplete(ctx); e != nil {
			err = e
		}
	}()

	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	// Use a preconfigured choice if it validates.
	chainID, err := l.j.pkg.PreconfiguredChoice(l.j.wl.ID)
	if err != nil {
		return uuid.Nil, err
	} else if chainID != uuid.Nil {
		// Fail if the choice is not available in workflow.
		var matched bool
		for _, cid := range l.config.Choices {
			if _, ok := l.j.wf.Chains[cid]; ok {
				matched = true
			}
		}
		if !matched {
			return uuid.Nil, fmt.Errorf("choice %s is not one of the available choices", chainID)
		}
		if err := l.j.markComplete(ctx); err != nil {
			return uuid.Nil, err
		}
		return chainID, nil
	}

	// Build decision point and await resolution.
	opts := make([]option, len(l.config.Choices))
	for i, item := range l.config.Choices {
		opts[i] = option(item.String())
	}

	return l.await(ctx, opts)
}

func (l *nextChainDecisionJob) await(ctx context.Context, opts []option) (_ uuid.UUID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("await: %v", err)
			return
		}
	}()

	if err := l.j.markAwaitingDecision(ctx); err != nil {
		return uuid.Nil, err
	}

	decision, err := l.j.pkg.AwaitDecision(ctx, opts)
	if err != nil {
		return uuid.Nil, err
	}

	return decision.uuid(), nil
}

// updateContextDecisionJob.
//
// A job that updates the job chain context based on a user choice.
//
// TODO: This type of job is mostly copied from the previous
// linkTaskManagerReplacementDicFromChoice, and it seems to have multiple ways
// of executing. It could use some cleanup.
//
// Manager: linkTaskManagerReplacementDicFromChoice.
// Class: UpdateContextDecisionJob(DecisionJob) (decisions.py).
type updateContextDecisionJob struct {
	j      *job
	config *workflow.LinkMicroServiceChoiceReplacementDic
}

var _ jobRunner = (*updateContextDecisionJob)(nil)

// Maps decision point UUIDs and decision UUIDs to their "canonical"
// equivalents. This is useful for when there are multiple decision points which
// are effectively identical and a preconfigured decision for one should hold
// for all of the others as well. For example, there are 5 "Assign UUIDs to
// directories?" decision points and making a processing config decision for the
// designated canonical one, in this case
// 'bd899573-694e-4d33-8c9b-df0af802437d', should result in that decision taking
// effect for all of the others as well. This allows that.
// TODO: this should be defined in the workflow, not hardcoded here.
var updateContextDecisionJobChoiceMapping = map[uuid.UUID]uuid.UUID{
	// Decision point "Assign UUIDs to directories?".
	uuid.MustParse("8882bad4-561c-4126-89c9-f7f0c083d5d7"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	uuid.MustParse("e10a31c3-56df-4986-af7e-2794ddfe8686"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	uuid.MustParse("d6f6f5db-4cc2-4652-9283-9ec6a6d181e5"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	uuid.MustParse("1563f22f-f5f7-4dfe-a926-6ab50d408832"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	// Decision "Yes" (for "Assign UUIDs to directories?").
	uuid.MustParse("7e4cf404-e62d-4dc2-8d81-6141e390f66f"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	uuid.MustParse("2732a043-b197-4cbc-81ab-4e2bee9b74d3"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	uuid.MustParse("aa793efa-1b62-498c-8f92-cab187a99a2a"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	uuid.MustParse("efd98ddb-80a6-4206-80bf-81bf00f84416"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	// Decision "No" (for "Assign UUIDs to directories?").
	uuid.MustParse("0053c670-3e61-4a3e-a188-3a2dd1eda426"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
	uuid.MustParse("8e93e523-86bb-47e1-a03a-4b33e13f8c5e"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
	uuid.MustParse("6dfbeff8-c6b1-435b-833a-ed764229d413"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
	uuid.MustParse("dc0ee6b6-ed5f-42a3-bc8f-c9c7ead03ed1"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
}

func newUpdateContextDecisionJob(j *job) (*updateContextDecisionJob, error) {
	config, ok := j.wl.Config.(workflow.LinkMicroServiceChoiceReplacementDic)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &updateContextDecisionJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *updateContextDecisionJob) exec(ctx context.Context) (linkID uuid.UUID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("nextChainDecisionJob: %v", err)
			return
		}
		if e := l.j.markComplete(ctx); e != nil {
			err = e
			return
		}
		if id := l.j.wl.ExitCodes[0].LinkID; id == nil || *id == uuid.Nil {
			err = fmt.Errorf("nextChainDecisionJob: linkID undefined")
		} else {
			linkID = *id
		}
	}()

	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	// Load new context from the database (DashboardSettings).
	// TODO: split this out? Workflow items with no replacements configured
	// seems like a different case.
	if len(l.config.Replacements) == 0 {
		if dict, err := l.loadDatabaseContext(ctx); err != nil {
			return uuid.Nil, fmt.Errorf("load dict from db: %v", err)
		} else if dict != nil {
			l.j.chain.update(dict)
			return uuid.Nil, nil
		}
	}

	// Load new context from processing configuration.
	if dict, err := l.loadPreconfiguredContext(); err != nil {
		return uuid.Nil, fmt.Errorf("load context with preconfigured choice: %v", err)
	} else if dict != nil {
		l.j.chain.update(dict)
		return uuid.Nil, nil
	}

	// Build decision point and await resolution.
	opts := make([]option, len(l.config.Replacements))
	for i, item := range l.config.Replacements {
		opts[i] = option(item.Description.String())
	}

	return l.await(ctx, opts)
}

// loadDatabaseContext loads the context dictionary from the database.
func (l *updateContextDecisionJob) loadDatabaseContext(ctx context.Context) (map[string]string, error) {
	ln, ok := l.j.wf.Links[l.j.wl.FallbackLinkID]
	if !ok {
		return nil, nil
	}
	cfg, ok := ln.Config.(workflow.LinkStandardTaskConfig)
	if !ok {
		return nil, nil
	}
	if cfg.Execute == "" {
		return nil, nil
	}

	ret, err := l.j.pkg.store.ReadDict(ctx, cfg.Execute)
	if err != nil {
		return nil, err
	}

	return l.formatChoices(ret), nil
}

// loadPreconfiguredContext loads the context dictionary from the workflow.
func (l *updateContextDecisionJob) loadPreconfiguredContext() (map[string]string, error) {
	var normalizedChoice uuid.UUID
	if v, ok := updateContextDecisionJobChoiceMapping[l.j.wl.ID]; ok {
		normalizedChoice = v
	} else {
		normalizedChoice = l.j.wl.ID
	}

	choices, err := l.j.pkg.parseProcessingConfig()
	if err != nil {
		return nil, err
	}

	for _, choice := range choices {
		if choice.AppliesTo != normalizedChoice.String() {
			continue
		}
		desiredChoice, err := uuid.Parse(choice.GoToChain)
		if err != nil {
			return nil, err
		}
		if v, ok := updateContextDecisionJobChoiceMapping[desiredChoice]; ok {
			desiredChoice = v
		}
		ln, ok := l.j.wf.Links[normalizedChoice]
		if !ok {
			return nil, nil // fmt.Errorf("desired choice not found: %s", desiredChoice)
		}
		config, ok := ln.Config.(workflow.LinkMicroServiceChoiceReplacementDic)
		if !ok {
			return nil, fmt.Errorf("desired choice doesn't have the expected type: %s", desiredChoice)
		}
		for _, replacement := range config.Replacements {
			if replacement.ID == desiredChoice.String() {
				choices := maps.Clone(replacement.Items)
				return l.formatChoices(choices), nil
			}
		}
	}

	return nil, nil
}

func (l *updateContextDecisionJob) formatChoices(choices map[string]string) map[string]string {
	for k, v := range choices {
		delete(choices, k)
		choices[fmt.Sprintf("%%%s%%", k)] = v
	}

	return choices
}

func (l *updateContextDecisionJob) await(ctx context.Context, opts []option) (_ uuid.UUID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("await: %v", err)
			return
		}
	}()

	if err := l.j.markAwaitingDecision(ctx); err != nil {
		return uuid.Nil, err
	}

	decision, err := l.j.pkg.AwaitDecision(ctx, opts) // nolint: staticcheck
	if err != nil {
		return uuid.Nil, err
	}

	// TODO: decision here should be an integer.
	// https://github.com/artefactual/archivematica/blob/2dd5a2366bf0529c193a19a5546087ed9a0b5534/src/MCPServer/lib/server/jobs/decisions.py#L286-L298

	panic("not implemented")

	return decision.uuid(), nil // nolint: govet
}

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
	config, ok := j.wl.Config.(workflow.LinkStandardTaskConfig)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &directoryClientScriptJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *directoryClientScriptJob) exec(ctx context.Context) (uuid.UUID, error) {
	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	taskResult, err := l.submitTasks(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}

	if err := l.j.updateStatusFromExitCode(ctx, taskResult.ExitCode); err != nil {
		return uuid.Nil, err
	}

	if ec, ok := l.j.wl.ExitCodes[taskResult.ExitCode]; ok {
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

func (l *directoryClientScriptJob) submitTasks(ctx context.Context) (*taskResult, error) {
	rm := l.j.pkg.unit.replacements(l.config.FilterSubdir).update(l.j.chain.pCtx)
	args := rm.replaceValues(l.config.Arguments)
	stdout := rm.replaceValues(l.config.StdoutFile)
	stderr := rm.replaceValues(l.config.StderrFile)

	taskBackend := newTaskBackend(l.j.logger, l.j, l.j.pkg.store, l.j.gearman, l.config)
	if err := taskBackend.submit(ctx, rm, args, false, stdout, stderr); err != nil {
		return nil, err
	}

	results, err := taskBackend.wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait: %v", err)
	}

	ret := results.First()
	if ret == nil {
		return nil, errors.New("submit task: no results")
	}

	return ret, nil
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
	config, ok := j.wl.Config.(workflow.LinkStandardTaskConfig)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &filesClientScriptJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *filesClientScriptJob) exec(ctx context.Context) (uuid.UUID, error) {
	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	filterSubDir, err := l.filterSubDir(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("look up filterSubDir: %v", err)
	}

	taskResults, err := l.submitTasks(ctx, filterSubDir)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}
	exitCode := 0
	if taskResults != nil {
		exitCode = taskResults.ExitCode()
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

	return l.j.wl.FallbackLinkID, nil
}

func (l *filesClientScriptJob) submitTasks(ctx context.Context, filterSubDir string) (*taskResults, error) {
	rm := l.j.pkg.unit.replacements(filterSubDir).update(l.j.chain.pCtx)
	taskBackend := newTaskBackend(l.j.logger, l.j, l.j.pkg.store, l.j.gearman, l.config)

	files, err := l.j.pkg.Files(ctx, l.config.FilterFileEnd, filterSubDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil // Nothing to do.
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
	config, ok := j.wl.Config.(workflow.LinkStandardTaskConfig)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &outputClientScriptJob{
		j:      j,
		config: &config,
	}, nil
}

// The list of choices are represented using a dictionary as follows:
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
	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	taskResult, err := l.submitTasks(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("submit task: %v", err)
	}

	choices := map[string]outputClientScriptChoice{}
	if err := json.Unmarshal([]byte(taskResult.Stdout), &choices); err != nil {
		l.j.logger.Error(err, "Unable to parse output: %s", taskResult.Stdout)
	} else {
		l.j.chain.choices = choices
	}

	if err := l.j.updateStatusFromExitCode(ctx, taskResult.ExitCode); err != nil {
		return uuid.Nil, err
	}

	if ec, ok := l.j.wl.ExitCodes[taskResult.ExitCode]; ok {
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

func (l *outputClientScriptJob) submitTasks(ctx context.Context) (*taskResult, error) {
	rm := l.j.pkg.unit.replacements(l.config.FilterSubdir).update(l.j.chain.pCtx)
	args := rm.replaceValues(l.config.Arguments)
	stdout := rm.replaceValues(l.config.StdoutFile)
	stderr := rm.replaceValues(l.config.StderrFile)

	taskBackend := newTaskBackend(l.j.logger, l.j, l.j.pkg.store, l.j.gearman, l.config)
	if err := taskBackend.submit(ctx, rm, args, true, stdout, stderr); err != nil {
		return nil, err
	}

	results, err := taskBackend.wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait: %v", err)
	}

	ret := results.First()
	if ret == nil {
		return nil, errors.New("submit task: no results")
	}

	return ret, nil
}

// setUnitVarLinkJob is a local job that sets the unit variable configured in
// the workflow.
//
// Manager: linkTaskManagerSetUnitVariable.
// Class: SetUnitVarLinkJob(DecisionJob) (decisions.py).
type setUnitVarLinkJob struct {
	j      *job
	config *workflow.LinkTaskConfigSetUnitVariable
}

var _ jobRunner = (*setUnitVarLinkJob)(nil)

func newSetUnitVarLinkJob(j *job) (*setUnitVarLinkJob, error) {
	config, ok := j.wl.Config.(workflow.LinkTaskConfigSetUnitVariable)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &setUnitVarLinkJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *setUnitVarLinkJob) exec(ctx context.Context) (uuid.UUID, error) {
	if err := l.j.pkg.saveLinkID(ctx, l.config.Variable, l.config.LinkID); err != nil {
		return uuid.Nil, err
	}

	if err := l.j.markComplete(ctx); err != nil {
		return uuid.Nil, err
	}

	return l.config.LinkID, nil
}

// getUnitVarLinkJob is a local job that gets the next link in the chain from a
// UnitVariable.
//
// Manager: linkTaskManagerUnitVariableLinkPull.
// Class: GetUnitVarLinkJob(DecisionJob) (decisions.py).
type getUnitVarLinkJob struct {
	j      *job
	config *workflow.LinkTaskConfigUnitVariableLinkPull
}

var _ jobRunner = (*getUnitVarLinkJob)(nil)

func newGetUnitVarLinkJob(j *job) (*getUnitVarLinkJob, error) {
	config, ok := j.wl.Config.(workflow.LinkTaskConfigUnitVariableLinkPull)
	if !ok {
		return nil, errors.New("invalid config")
	}

	return &getUnitVarLinkJob{
		j:      j,
		config: &config,
	}, nil
}

func (l *getUnitVarLinkJob) exec(ctx context.Context) (uuid.UUID, error) {
	if err := l.j.pkg.reload(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reload: %v", err)
	}
	if err := l.j.save(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("save: %v", err)
	}

	linkID, err := l.j.pkg.store.ReadUnitLinkID(ctx, l.j.pkg.id, l.j.pkg.packageType(), l.config.Variable)
	if errors.Is(err, store.ErrNotFound) {
		return l.config.LinkID, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("read: %v", err)
	}
	if linkID == uuid.Nil {
		linkID = l.config.LinkID
	}

	if err := l.j.markComplete(ctx); err != nil {
		return uuid.Nil, err
	}

	return linkID, nil
}
