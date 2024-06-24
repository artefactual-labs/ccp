package controller

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

var errEnd = errors.New("terminator")

type jobIterator struct {
	ctx      context.Context
	logger   logr.Logger
	metrics  *metrics.Metrics
	gearman  *gearmin.Server
	wf       *workflow.Document
	pkg      *Package
	nextLink uuid.UUID // Next workflow link or workflow chain link.
	chain    *chain    // Current workflow chain
}

func newJobIterator(ctx context.Context, logger logr.Logger, metrics *metrics.Metrics, gearman *gearmin.Server, wf *workflow.Document, pkg *Package) *jobIterator {
	iter := &jobIterator{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
		gearman: gearman,
		wf:      wf,
		pkg:     pkg,
	}

	return iter
}

func (i *jobIterator) init() error {
	i.logger.Info("Init iterator.")

	err := i.pkg.markAsProcessing(i.ctx)
	if err != nil {
		return err
	}

	wc, ok := i.wf.Chains[i.pkg.startAtChainID]
	if !ok {
		return fmt.Errorf("can't process a job without a chain")
	} else {
		i.nextLink = wc.ID // Starting point.
	}

	return nil
}

func (i *jobIterator) next() error {
	if err := i.ctx.Err(); err != nil {
		return err
	}

	i.logger.Info("Starting new iteration.", "nextLink", i.nextLink)

	// Only when we start the iterator.
	if i.nextLink == uuid.Nil {
		if err := i.init(); err != nil {
			return err
		}
	}

	if wc, ok := i.wf.Chains[i.nextLink]; ok {
		i.logger.Info("Starting new chain.", "id", wc.ID, "desc", wc.Description)
		i.chain = newChain(wc)
		if err := i.chain.load(i.ctx, i.pkg); err != nil {
			return fmt.Errorf("load context: %v", err)
		}
		// Special case where the next list is override with the bypass.
		if wc.ID == i.pkg.startAtChainID && i.pkg.startAtLinkID != uuid.Nil {
			i.nextLink = i.pkg.startAtLinkID
		} else {
			i.nextLink = wc.LinkID // Normal flow.
		}
		return nil
	}

	if i.chain == nil {
		return fmt.Errorf("can't process a job without a chain")
	}

	wl, ok := i.wf.Links[i.nextLink]
	if !ok {
		return fmt.Errorf("link not found in workflow document")
	}

	j, err := i.buildJob(wl, i.logger.WithName("job"))
	if err != nil {
		return fmt.Errorf("build job for link %s: %v", wl.ID, err)
	}

	next, err := j.exec(i.ctx)
	j.logger.Info("Job executed.", "name", j.wl.Description, "err", err)

	if errors.Is(err, io.EOF) {
		if wl.End {
			if err := j.pkg.markAsDone(i.ctx); err != nil {
				j.logger.Error(err, "Failed to mark the package as done.")
			}
			return errEnd
		} else {
			// Signal end of this iterator.
			// Workflow must continue using a watched directory.
			//
			// TODO: continue work in this iterator.
			return io.EOF
		}
	} else if _, ok := isErrWait(err); ok {
		return err
	} else if err != nil {
		if err := j.pkg.markAsFailed(i.ctx); err != nil {
			j.logger.Error(err, "Failed to mark the package as failed.")
		}
		return fmt.Errorf("exec job for link %s with manager %s (%s) : %v", wl.ID, wl.Manager, wl.Description, err)
	}

	i.nextLink = next

	return nil
}

// buildJob configures a workflow job given the workflow chain link definition.
func (i *jobIterator) buildJob(wl *workflow.Link, logger logr.Logger) (*job, error) {
	logger = logger.WithValues(
		"type", "link",
		"linkID", wl.ID,
		"desc", wl.Description,
		"manager", wl.Manager,
		"terminator", wl.End,
	)

	j, err := newJob(logger, i.metrics, i.chain, i.pkg, i.gearman, wl, i.wf)
	if err != nil {
		return nil, fmt.Errorf("build job: %v", err)
	}

	return j, nil
}
