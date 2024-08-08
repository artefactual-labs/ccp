package controller

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

func TestNextChainDecisionJob(t *testing.T) {
	t.Parallel()

	t.Run("Honours preconfigured choices", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "56eebd45-5600-4768-a8c2-ec0114555a3d")
		createAutomatedProcessingConfig(t, job.pkg.path)

		store.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.MustParse("e9eaef1e-c2e0-4e3b-b942-bfb537162795"))
		assert.NilError(t, err)
	})

	t.Run("Creates a decision", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "56eebd45-5600-4768-a8c2-ec0114555a3d")

		store.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.Nil)

		decision := assertErrWait(t, err, "Generate transfer structure report", []choice{
			{label: "Yes", nextLink: uuid.MustParse("df54fec1-dae1-4ea6-8d17-a839ee7ac4a7")},
			{label: "No", nextLink: uuid.MustParse("e9eaef1e-c2e0-4e3b-b942-bfb537162795")},
		})

		decision.resolveWithPos(0)
		nextLink, err := decision.await(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, nextLink, uuid.MustParse("df54fec1-dae1-4ea6-8d17-a839ee7ac4a7"))
	})

	t.Run("Excludes choices related to disabled abilities", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "bb194013-597c-4e4a-8493-b36d190f8717")

		store.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.Nil)

		decision := assertErrWait(t, err, "Create SIP(s)", []choice{
			{label: "Create single SIP and continue processing", nextLink: uuid.MustParse("61cfa825-120e-4b17-83e6-51a42b67d969")},
			{label: "Reject transfer", nextLink: uuid.MustParse("1b04ec43-055c-43b7-9543-bd03c6a778ba")},
		})

		decision.resolveWithPos(0)
		nextLink, err := decision.await(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, nextLink, uuid.MustParse("61cfa825-120e-4b17-83e6-51a42b67d969"))
	})

	t.Run("Resolves decision when the position is unknown", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "bb194013-597c-4e4a-8493-b36d190f8717")

		store.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.Nil)

		decision := assertErrWait(t, err, "Create SIP(s)", []choice{
			{label: "Create single SIP and continue processing", nextLink: uuid.MustParse("61cfa825-120e-4b17-83e6-51a42b67d969")},
			{label: "Reject transfer", nextLink: uuid.MustParse("1b04ec43-055c-43b7-9543-bd03c6a778ba")},
		})

		decision.resolveWithChoice("61cfa825-120e-4b17-83e6-51a42b67d969")
		nextLink, err := decision.await(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, nextLink, uuid.MustParse("61cfa825-120e-4b17-83e6-51a42b67d969"))
	})
}

func TestUpdateContextDecisionJob(t *testing.T) {
	t.Parallel()

	t.Run("Honours database context", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "a0db8294-f02a-4f49-a557-b1310a715ffc")

		store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(mockutil.Context(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		store.EXPECT().ReadDict(mockutil.Context(), "upload-archivesspace_v0.0").Return(
			map[string]string{
				"username": "test",
				"password": "test",
			},
			nil,
		)

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.MustParse("ff89a530-0540-4625-8884-5a2198dea05a"))
		assert.NilError(t, err)

		assertChainContext(t, job.chain, map[string]string{
			"%username%": "test",
			"%password%": "test",
		})
	})

	t.Run("Honours preconfigured choices", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "8882bad4-561c-4126-89c9-f7f0c083d5d7")
		createAutomatedProcessingConfig(t, job.pkg.path)

		store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(mockutil.Context(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.MustParse("5415c813-3637-49ab-afec-9b435c2e4d2c"))
		assert.NilError(t, err)

		assertChainContext(t, job.chain, map[string]string{
			"%AssignUUIDsToDirectories%": "True",
		})
	})

	t.Run("Creates a decision", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "8882bad4-561c-4126-89c9-f7f0c083d5d7")

		store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(mockutil.Context(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.Nil)

		decision := assertErrWait(t, err, "Assign UUIDs to directories?", []choice{
			{label: "No", value: [2]string{"AssignUUIDsToDirectories", "False"}, nextLink: uuid.MustParse("5415c813-3637-49ab-afec-9b435c2e4d2c")},
			{label: "Yes", value: [2]string{"AssignUUIDsToDirectories", "True"}, nextLink: uuid.MustParse("5415c813-3637-49ab-afec-9b435c2e4d2c")},
		})

		decision.resolveWithPos(0)
		nextLink, err := decision.await(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, nextLink, uuid.MustParse("5415c813-3637-49ab-afec-9b435c2e4d2c"))

		assertChainContext(t, job.chain, map[string]string{
			"%AssignUUIDsToDirectories%": "False",
		})
	})
}

func assertErrWait(t *testing.T, err error, name string, choices []choice) *decision {
	t.Helper()

	ew := &errWait{}
	assert.Equal(t, errors.As(err, &ew), true)

	assert.Equal(t, ew.decision.name, name)
	assert.DeepEqual(t, ew.decision.choices, choices, cmpopts.EquateComparable(choice{}))

	return ew.decision
}

func assertChainContext(t *testing.T, c *chain, expected map[string]string) {
	t.Helper()

	data := map[string]string{}
	for el := c.context.Front(); el != nil; el = el.Next() {
		data[el.Key] = el.Value
	}

	assert.DeepEqual(t, data, expected)
}

func createAutomatedProcessingConfig(t *testing.T, path string) {
	t.Helper()

	path = filepath.Join(path, "processingMCP.xml")

	err := workflow.SaveConfigFile(path, workflow.AutomatedConfig.Choices)
	assert.NilError(t, err)
}
