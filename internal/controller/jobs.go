package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/derrors"
	"github.com/artefactual-labs/ccp/internal/store/enums"
	"github.com/artefactual-labs/ccp/internal/store/sqlcmysql"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

// job is the representation of a single execution of a workflow link. It keeps
// references to the workflow document and the package that is being processed,
// as well as the dispatcher and the store. A *job implements the jobRunner
// interface, which means it can be executed.
type job struct {
	logger  logr.Logger
	metrics *metrics.Metrics

	// dispatcher handles sending tasks to workers.
	dispatcher *dispatcher.Dispatcher

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

	// finalStatusRecorded prevents multiple updates to the job's final status
	// which can occur if both updateStatusFromExitCode and markComplete are
	// called.
	finalStatusRecorded bool
}

// jobRunner is the interface that all jobs must implement.
type jobRunner interface {
	exec(context.Context) (uuid.UUID, error)
}

func newJob(logger logr.Logger, metrics *metrics.Metrics, chain *chain, pkg *Package, dispatcher *dispatcher.Dispatcher, wl *workflow.Link, wf *workflow.Document) (*job, error) {
	j := &job{
		logger:     logger,
		metrics:    metrics,
		dispatcher: dispatcher,
		id:         uuid.New(),
		createdAt:  time.Now().UTC(),
		chain:      chain,
		pkg:        pkg,
		wl:         wl,
		wf:         wf,
	}

	var err error
	switch wl.Manager {

	// Decision jobs - handles workflow decision points.
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

func (j *job) exec(ctx context.Context) (id uuid.UUID, err error) {
	defer derrors.Wrap(&err, "exec")

	if err := j.save(ctx); err != nil {
		return uuid.Nil, err
	}

	id, err = j.jobRunner.exec(ctx)

	if _, ok := isErrWait(err); ok {
		if markErr := j.markAwaitingDecision(ctx); markErr != nil {
			err = markErr
		}
	} else if err == nil {
		if markErr := j.markComplete(ctx); markErr != nil {
			err = markErr
		}
	}

	return id, err
}

// save the job in the store.
func (j *job) save(ctx context.Context) (err error) {
	defer derrors.Add(&err, "save")

	// Reload the package before creating the job.
	if err := j.pkg.reload(ctx); err != nil {
		return fmt.Errorf("reload package: %v", err)
	}

	return j.pkg.store.CreateJob(ctx, &sqlcmysql.CreateJobParams{
		ID:                j.id,
		Type:              j.wl.Description.String(),
		CreatedAt:         j.createdAt,
		Directory:         j.pkg.PathForDB(),
		SIPID:             j.pkg.id,
		Unittype:          j.pkg.jobUnitType(),
		Currentstep:       int32(enums.JobStatusExecutingCommands),
		Microservicegroup: j.wl.Group.String(),
		Hidden:            false,
		LinkID: uuid.NullUUID{
			UUID:  j.wl.ID,
			Valid: true,
		},
	})
}

// markAwaitingDecision is used by decision jobs to persist the awaiting status.
func (j *job) markAwaitingDecision(ctx context.Context) error {
	err := j.pkg.store.UpdateJobStatus(ctx, j.id, enums.JobStatusAwaitingDecision)
	if err != nil {
		return fmt.Errorf("mark awaiting decision: %v", err)
	}

	return nil
}

// markComplete is used by decision jobs to persist the completion status.
func (j *job) markComplete(ctx context.Context) error {
	// Certain jobs may have already used updateStatusFromExitCode.
	if j.finalStatusRecorded {
		return nil
	}

	err := j.pkg.store.UpdateJobStatus(ctx, j.id, enums.JobStatusCompletedSuccessfully)
	if err != nil {
		return fmt.Errorf("mark complete: %v", err)
	}

	j.finalStatusRecorded = true

	return nil
}

// updateStatusFromExitCode is used by client jobs to persist the status.
func (j *job) updateStatusFromExitCode(ctx context.Context, code int) error {
	status := ""
	if ec, ok := j.wl.ExitCodes[code]; ok {
		status = ec.JobStatus
	} else {
		status = j.wl.FallbackJobStatus
	}

	// Convert statuses from workflow.
	jobStatus := enums.JobStatusUnknown
	switch status {
	case "Completed successfully":
		jobStatus = enums.JobStatusCompletedSuccessfully
	case "Failed":
		jobStatus = enums.JobStatusFailed
	}

	err := j.pkg.store.UpdateJobStatus(ctx, j.id, jobStatus)
	if err != nil {
		return fmt.Errorf("update job status from exit code: %v", err)
	}

	j.finalStatusRecorded = true

	return nil
}

// exitCodeLinkID returns the next link ID to execute based on the exit code.
func exitCodeLinkID(l *workflow.Link, code int) uuid.UUID {
	ret := uuid.Nil

	if ec, ok := l.ExitCodes[code]; ok {
		if ec.LinkID != nil {
			ret = *ec.LinkID
		}
	}

	if ret == uuid.Nil {
		ret = l.FallbackLinkID
	}

	return ret
}

type ConfigT interface {
	workflow.LinkStandardTaskConfig |
		workflow.LinkTaskConfigSetUnitVariable |
		workflow.LinkTaskConfigUnitVariableLinkPull |
		workflow.LinkMicroServiceChainChoice |
		workflow.LinkMicroServiceChoiceReplacementDic
}

// loadConfig loads the configuration from the workflow link into the provided
// destination variable.
func loadConfig[T ConfigT](wl *workflow.Link, dest *T) error {
	config, ok := wl.Config.(T)
	if !ok {
		return fmt.Errorf("config provided is not compatible with its type")
	}

	*dest = config

	return nil
}
