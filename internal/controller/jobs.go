package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/derrors"
	"github.com/artefactual-labs/ccp/internal/store/sqlcmysql"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

type job struct {
	logger  logr.Logger
	metrics *metrics.Metrics

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

	// finalStatusRecorded remembers if updateStatusFromExitCode was used.
	finalStatusRecorded bool
}

// jobRunner is the interface that all jobs must implement.
type jobRunner interface {
	exec(context.Context) (uuid.UUID, error)
}

func newJob(logger logr.Logger, metrics *metrics.Metrics, chain *chain, pkg *Package, gearman *gearmin.Server, wl *workflow.Link, wf *workflow.Document) (*job, error) {
	j := &job{
		logger:    logger,
		metrics:   metrics,
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
		Createdtimedec:    fmt.Sprintf("%.9f", float64(j.createdAt.Nanosecond())/1e9),
		Directory:         j.pkg.PathForDB(),
		SIPID:             j.pkg.id,
		Unittype:          j.pkg.jobUnitType(),
		Currentstep:       3,
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
	err := j.pkg.store.UpdateJobStatus(ctx, j.id, "STATUS_AWAITING_DECISION")
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

	err := j.pkg.store.UpdateJobStatus(ctx, j.id, "STATUS_COMPLETED_SUCCESSFULLY")
	if err != nil {
		return fmt.Errorf("mark complete: %v", err)
	}

	j.finalStatusRecorded = true

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

	j.finalStatusRecorded = true

	return nil
}

// processTasksResults processes a set of task results produced by a client job,
// e.g.: filesClientScriptJob. It returns the highest exist code seen.
func (j *job) processTaskResults(cfg *workflow.LinkStandardTaskConfig, tr *taskResults) int {
	maxExitCode := 0

	for _, result := range tr.Results {
		j.metrics.TaskCompleted(
			result.task.CreatedAt,
			result.FinishedAt,
			cfg.Execute,
			j.wl.Group.String(),
			j.wl.Description.String(),
		)

		// Calculate the maximum exit code.
		if result.ExitCode > maxExitCode {
			maxExitCode = result.ExitCode
		}
	}

	return maxExitCode
}

func exitCodeLinkID(l *workflow.Link, code int) uuid.UUID { //nolint:unparam
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

func loadConfig[T ConfigT](wl *workflow.Link, dest *T) error {
	config, ok := wl.Config.(T)
	if !ok {
		return fmt.Errorf("config provided is not compatible with its type")
	}

	*dest = config

	return nil
}
