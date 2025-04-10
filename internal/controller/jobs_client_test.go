package controller

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/store"
	"github.com/artefactual-labs/ccp/internal/store/enums"
)

func TestDirectoryClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Run("Runs a task and handles its success outcome", func(t *testing.T) {
		t.Parallel()

		tj := createJob(t, "002716a1-ae29-4f36-98ab-0d97192669c4") // Move to compressionAIPDecisions directory.
		createAutomatedProcessingConfig(t, tj.job.pkg.path)

		tj.store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		tj.store.EXPECT().UpdateJobStatus(mockutil.Context(), tj.job.id, enums.JobStatusCompletedSuccessfully).Return(nil).Times(1)

		gomock.InOrder(
			tj.backend.EXPECT().Name().Return("workhub").Times(1),
			tj.backend.EXPECT().
				Dispatch(mockutil.Context(), tj.job.id, "moveSIP_v0.0", gomock.Any()).
				Return([]*dispatcher.TaskResult{
					{
						ExitCode: 0,
					},
				}, nil).Times(1),
		)

		_, ok := tj.job.jobRunner.(*directoryClientScriptJob)
		assert.Assert(t, ok)

		nextLinkID, err := tj.job.exec(t.Context())
		assert.Equal(t, nextLinkID, uuid.Nil)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Runs a task and handles its failed outcome", func(t *testing.T) {
		t.Parallel()

		tj := createJob(t, "002716a1-ae29-4f36-98ab-0d97192669c4") // Move to compressionAIPDecisions directory.
		createAutomatedProcessingConfig(t, tj.job.pkg.path)

		tj.store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		tj.store.EXPECT().UpdateJobStatus(mockutil.Context(), tj.job.id, enums.JobStatusFailed).Return(nil).Times(1)

		gomock.InOrder(
			tj.backend.EXPECT().Name().Return("workhub").Times(1),
			tj.backend.EXPECT().
				Dispatch(mockutil.Context(), tj.job.id, "moveSIP_v0.0", gomock.Any()).
				Return([]*dispatcher.TaskResult{
					{
						ExitCode: 1,
					},
				}, nil).Times(1),
		)

		_, ok := tj.job.jobRunner.(*directoryClientScriptJob)
		assert.Assert(t, ok)

		nextLinkID, err := tj.job.exec(t.Context())
		assert.NilError(t, err)
		assert.Equal(t, nextLinkID, uuid.MustParse("7d728c39-395f-4892-8193-92f086c0546f"))
	})
}

func TestFilesClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Run("Runs multiple task and handles its success outcome", func(t *testing.T) {
		t.Parallel()

		tj := createJob(t, "0e41c244-6c3e-46b9-a554-65e66e5c9324") // Identify file format of attachments.
		createAutomatedProcessingConfig(t, tj.job.pkg.path)

		tj.store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		tj.store.EXPECT().UpdateJobStatus(mockutil.Context(), tj.job.id, enums.JobStatusCompletedSuccessfully).Return(nil).Times(1)
		tj.store.EXPECT().ReadUnitVar(mockutil.Context(), tj.job.pkg.id, enums.PackageTypeTransfer, "identifyFileFormat_v0.0").Return("", nil).Times(1)
		tj.store.EXPECT().Files(mockutil.Context(), tj.job.pkg.id, enums.PackageTypeTransfer, "", "objects/attachments", "").Return([]store.File{
			{
				ID:               uuid.MustParse("c495f089-896b-44f5-b0ac-5226625abd9b"),
				CurrentLocation:  "f1",
				OriginalLocation: "f1",
			},
			{
				ID:               uuid.MustParse("6f2c3b0f-fbab-45ba-8eb0-acb2add65c63"),
				CurrentLocation:  "f2",
				OriginalLocation: "f2",
			},
		}, nil).Times(1)

		gomock.InOrder(
			tj.backend.EXPECT().Name().Return("workhub").Times(1),
			tj.backend.EXPECT().
				Dispatch(mockutil.Context(), tj.job.id, "identifyFileFormat_v0.0",
					mockutil.Func("", func(tasks []*dispatcher.Task) error {
						assert.Equal(t, len(tasks), 2)
						return nil
					}),
				).
				Return([]*dispatcher.TaskResult{
					{
						ExitCode: 0,
					},
					{
						ExitCode: 0,
					},
				}, nil).Times(1),
		)

		attachmentsDir := filepath.Join(tj.job.pkg.path, "objects/attachments")
		_ = os.MkdirAll(attachmentsDir, os.FileMode(0o755))
		_, _ = os.CreateTemp(attachmentsDir, "f1-*")
		_, _ = os.CreateTemp(attachmentsDir, "f2-*")

		_, ok := tj.job.jobRunner.(*filesClientScriptJob)
		assert.Assert(t, ok)

		// TODO: verify tasks requested to the dispatcher.

		nextLinkID, err := tj.job.exec(t.Context())
		assert.Equal(t, nextLinkID, uuid.MustParse("95616c10-a79f-48ca-a352-234cc91eaf08"))
		assert.NilError(t, err)
	})
}
