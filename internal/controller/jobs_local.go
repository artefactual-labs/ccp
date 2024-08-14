package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/artefactual-labs/ccp/internal/derrors"
	"github.com/artefactual-labs/ccp/internal/store"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

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
	ret := &setUnitVarLinkJob{
		j:      j,
		config: &workflow.LinkTaskConfigSetUnitVariable{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *setUnitVarLinkJob) exec(ctx context.Context) (_ uuid.UUID, err error) {
	derrors.Add(&err, "setUnitVarLinkJob")

	if err := l.j.pkg.saveLinkID(ctx, l.config.Variable, l.config.LinkID); err != nil {
		return uuid.Nil, err
	}

	return exitCodeLinkID(l.j.wl, 0), nil
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
	ret := &getUnitVarLinkJob{
		j:      j,
		config: &workflow.LinkTaskConfigUnitVariableLinkPull{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *getUnitVarLinkJob) exec(ctx context.Context) (_ uuid.UUID, err error) {
	derrors.Add(&err, "getUnitVarLinkJob")

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

	return linkID, nil
}
