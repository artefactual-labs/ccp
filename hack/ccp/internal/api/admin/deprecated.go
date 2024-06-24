package admin

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.artefactual.dev/tools/ref"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/controller"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
)

var approveAIPReingestChainID = uuid.MustParse("260ef4ea-f87d-4acf-830d-d0de41e6d2af")

// ApproveJob is obsolote and exists for backward-compatibility purposes.
func (s *Server) ApproveJob(ctx context.Context, req *connect.Request[adminv1.ApproveJobRequest]) (*connect.Response[adminv1.ApproveJobResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	jobID := uuid.MustParse(req.Msg.JobId)
	err := s.ctrl.ResolveDecisionLegacy(jobID, req.Msg.Choice)
	if err != nil {
		s.logger.V(2).Info("Failed to approve job.", "err", err, "jobID", jobID, "choice", req.Msg.Choice)
		return nil, connect.NewError(connect.CodeUnknown, nil)
	}

	return connect.NewResponse(&adminv1.ApproveJobResponse{}), nil
}

// ApproveTransferByPath is obsolete and exists for backward-compatibility purposes.
func (s *Server) ApproveTransferByPath(ctx context.Context, req *connect.Request[adminv1.ApproveTransferByPathRequest]) (*connect.Response[adminv1.ApproveTransferByPathResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	transferType := req.Msg.Type
	if transferType == adminv1.TransferType_TRANSFER_TYPE_UNSPECIFIED {
		transferType = adminv1.TransferType_TRANSFER_TYPE_STANDARD
	}

	var reqDir string
	dir, file := filepath.Split(req.Msg.Directory)
	if dir != "" && file == "" {
		// When the Dashboard prepares it, e.g.:
		// "%sharedPath%watchedDirectories/activeTransfers/standardTransfer/tmp.CuWEWmfl09/"
		reqDir = dir
	} else if dir == "" && file != "" {
		tt := controller.Transfers.WithType(transferType)
		if tt == nil {
			return nil, connect.NewError(connect.CodeUnknown, errors.New("unexpected transfer type"))
		}
		reqDir = fmt.Sprintf("%%sharedPath%%watchedDirectories/%s/%s/", tt.WatchedDir, file)
	} else {
		return nil, connect.NewError(connect.CodeUnknown, errors.New("unexpected directory"))
	}

	job, err := s.store.FindAwaitingJob(ctx, &store.FindAwaitingJobParams{Directory: &reqDir})
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("unable to find awaiting job: %s", reqDir))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("unable to find awaiting job: %s", reqDir))
	}

	jobID, _ := uuid.Parse(job.Id)
	pkgID, _ := uuid.Parse(job.PackageId)

	var chainID uuid.UUID
	if t := controller.Transfers.WithType(transferType); t == nil {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("unable to resolve for transfer type: %s", transferType.String()))
	} else {
		chainID = t.BypassChainID
	}

	if err := s.ctrl.ResolveDecisionLegacy(jobID, chainID.String()); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("unable to resolve decision: %v", err))
	}

	return connect.NewResponse(&adminv1.ApproveTransferByPathResponse{
		Id: pkgID.String(),
	}), nil
}

// ApprovePartialReingest is obsolete and exists for backward-compatibility purposes.
func (s *Server) ApprovePartialReingest(ctx context.Context, req *connect.Request[adminv1.ApprovePartialReingestRequest]) (*connect.Response[adminv1.ApprovePartialReingestResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	pkgID, _ := uuid.Parse(req.Msg.Id)
	job, err := s.store.FindAwaitingJob(ctx, &store.FindAwaitingJobParams{
		PackageID: &pkgID,
		Group:     ref.New("Reingest AIP"),
	})
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("unable to find awaiting job: %s", req.Msg.Id))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("unable to find awaiting job: %s", req.Msg.Id))
	}

	chain, ok := s.wf.Chains[approveAIPReingestChainID]
	if !ok {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("unable to find reingest chain: %s", approveAIPReingestChainID.String()))
	}

	jobID, _ := uuid.Parse(job.Id)
	if err := s.ctrl.ResolveDecisionLegacy(jobID, chain.ID.String()); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("unable to resolve decision: %v", err))
	}

	return connect.NewResponse(&adminv1.ApprovePartialReingestResponse{}), nil
}
