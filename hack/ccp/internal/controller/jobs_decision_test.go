package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestOutputDecisionJob(t *testing.T) {
	t.Parallel()
}

func TestNextChainDecisionJob(t *testing.T) {
	t.Parallel()

	t.Run("Honours preconfigured choices", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("Creates a decision", func(t *testing.T) {
		t.Parallel()

		job, store := createJob(t, "56eebd45-5600-4768-a8c2-ec0114555a3d")

		store.EXPECT().UpdateJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		store.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		id, err := job.exec(context.Background())
		assert.Equal(t, id, uuid.Nil)

		decision := assertErrWait(t, err, "Generate transfer structure report", []choice{
			{label: "Yes", nextLink: uuid.MustParse("df54fec1-dae1-4ea6-8d17-a839ee7ac4a7")},
			{label: "No", nextLink: uuid.MustParse("e9eaef1e-c2e0-4e3b-b942-bfb537162795")},
		})

		decision.resolve(0)
		nextLink, err := decision.await(context.Background())
		assert.NilError(t, err)
		assert.Equal(t, nextLink, uuid.MustParse("df54fec1-dae1-4ea6-8d17-a839ee7ac4a7"))
	})
}

func TestUpdateContextDecisionJob(t *testing.T) {
	t.Parallel()
}

func assertErrWait(t *testing.T, err error, name string, choices []choice) *decision {
	t.Helper()

	ew := &errWait{}
	assert.Equal(t, errors.As(err, &ew), true)

	assert.Equal(t, ew.decision.name, name)
	assert.DeepEqual(t, ew.decision.choices, choices, cmpopts.EquateComparable(choice{}))

	return ew.decision
}
