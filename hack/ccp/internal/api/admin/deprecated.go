package admin

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

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

	// 1. Find the first Job with (directory=req.Msg.Directory, currentstep="Job.STATUS_AWAITING_DECISION")
	// 2. Find the chain ID to approve the transfer from the req.Msg.TransferType.
	// 3. Resolve deicision using the ID of the job and the ID of the chain.
	// If unmatched, return connect.NewError(connect.CodeNotFound, nil).
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// ApprovePartialReingest is obsolete and exists for backward-compatibility purposes.
func (s *Server) ApprovePartialReingest(ctx context.Context, req *connect.Request[adminv1.ApprovePartialReingestRequest]) (*connect.Response[adminv1.ApprovePartialReingestResponse], error) {
	if err := s.v.Validate(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// 1. Find the first Job with (sipuuid=req.Msg.Id, microservicegroup="Reingest AIP", currentstep="Job.STATUS_AWAITING_DECIISON")
	// 2. Resolve decision using the ID of the job and the ID of the chain ("260ef4ea-f87d-4acf-830d-d0de41e6d2af")
	// 3. If unmatched, return connect.NewError(connect.CodeNotFound, nil).
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
