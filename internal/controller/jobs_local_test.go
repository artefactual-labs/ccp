package controller

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/store/enums"
	"github.com/artefactual-labs/ccp/internal/store/sqlcmysql"
)

func TestSetUnitVarLinkJob(t *testing.T) {
	t.Parallel()

	t.Run("Stores the linkID in the database", func(t *testing.T) {
		t.Parallel()

		job, st := createJob(t, "b33c9544-145c-4525-8a80-d686b4d1c3fa")

		st.EXPECT().CreateJob(mockutil.Context(), gomock.AssignableToTypeOf(&sqlcmysql.CreateJobParams{})).Return(nil).Times(1)
		st.EXPECT().CreateUnitVar(mockutil.Context(), job.pkg.id, enums.PackageTypeTransfer, "normalizationThumbnailProcessing", "", uuid.MustParse("180ae3d0-aa6c-4ed4-ab94-d0a2121e7f21"), true).Times(1)
		st.EXPECT().UpdateJobStatus(mockutil.Context(), job.id, "STATUS_COMPLETED_SUCCESSFULLY").Return(nil).Times(1)

		linkID, err := job.exec(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, linkID, uuid.MustParse("498f7a6d-1b8c-431a-aa5d-83f14f3c5e65"))
	})
}

func TestGetUnitVarLinkJob(t *testing.T) {
	t.Parallel()

	t.Run("Returns the linkID stored in the database", func(t *testing.T) {
		t.Parallel()

		nextLinkID := uuid.MustParse("0ce7ab48-fd48-4abe-9150-f682499e7cf0")
		job, st := createJob(t, "6e5126be-76ac-4c8f-9754-fc25a234a751")

		st.EXPECT().CreateJob(mockutil.Context(), gomock.AssignableToTypeOf(&sqlcmysql.CreateJobParams{})).Return(nil).Times(1)
		st.EXPECT().UpdateJobStatus(mockutil.Context(), job.id, "STATUS_COMPLETED_SUCCESSFULLY").Return(nil).Times(1)
		st.EXPECT().ReadUnitLinkID(mockutil.Context(), job.pkg.id, enums.PackageTypeTransfer, "normalizationThumbnailProcessing").Return(nextLinkID, nil).Times(1)

		linkID, err := job.exec(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, linkID, nextLinkID)
	})

	t.Run("Returns the linkID defined in workflow", func(t *testing.T) {
		t.Parallel()

		job, st := createJob(t, "b04e9232-2aea-49fc-9560-27349c8eba4e")

		st.EXPECT().CreateJob(mockutil.Context(), gomock.AssignableToTypeOf(&sqlcmysql.CreateJobParams{})).Return(nil).Times(1)
		st.EXPECT().UpdateJobStatus(mockutil.Context(), job.id, "STATUS_COMPLETED_SUCCESSFULLY").Return(nil).Times(1)
		st.EXPECT().ReadUnitLinkID(mockutil.Context(), job.pkg.id, enums.PackageTypeTransfer, "loadOptionsToCreateSIP").Return(uuid.MustParse("bb194013-597c-4e4a-8493-b36d190f8717"), nil).Times(1)

		linkID, err := job.exec(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, linkID, uuid.MustParse("bb194013-597c-4e4a-8493-b36d190f8717"))
	})
}
