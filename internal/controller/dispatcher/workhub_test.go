package dispatcher_test

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp/cmpopts"
	uuid "github.com/google/uuid"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher/dispatchermock"
	"github.com/artefactual-labs/ccp/internal/store/storemock"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

func TestWorkhubDispatcher(t *testing.T) {
	t.Parallel()

	logger := logr.Discard()
	metrics := metrics.NewMetrics(nil)
	ctrl := gomock.NewController(t)
	storeMock := storemock.NewMockStore(ctrl)
	mockBackend := dispatchermock.NewMockBackend(ctrl)

	jobID := uuid.New()

	// Create dispatcher with mocked backend
	dis := dispatcher.NewWithBackend(logger, metrics, storeMock, mockBackend)

	// Set up mock expectations
	mockBackend.EXPECT().Name().Return("test-backend").AnyTimes()

	mockBackend.EXPECT().
		Dispatch(
			mockutil.Context(),
			jobID,
			"test_function",
			gomock.Any(),
		).
		Times(1).
		Return([]*dispatcher.TaskResult{
			{
				ExitCode: 0,
			},
		}, nil)

	res, err := dis.Dispatch(
		t.Context(),
		jobID,
		&workflow.LinkStandardTaskConfig{
			Execute: "test_function",
		},
		[]*dispatcher.TaskRequest{
			{},
		},
	)
	assert.NilError(t, err)

	assert.DeepEqual(t, res,
		[]*dispatcher.TaskResult{
			{
				ExitCode: 0,
			},
		},
		cmpopts.IgnoreFields(dispatcher.TaskResult{}, "CreatedAt", "FinishedAt"),
	)
}

func TestWorkhubDispatcherWithOutput(t *testing.T) {
	t.Parallel()

	logger := logr.Discard()
	metrics := metrics.NewMetrics(nil)
	ctrl := gomock.NewController(t)
	storeMock := storemock.NewMockStore(ctrl)
	mockBackend := dispatchermock.NewMockBackend(ctrl)

	jobID := uuid.New()

	dis := dispatcher.NewWithBackend(logger, metrics, storeMock, mockBackend)

	// Create a temporary directory for output files
	tmpDir := t.TempDir()
	stdoutFile := tmpDir + "/stdout.txt"
	stderrFile := tmpDir + "/stderr.txt"

	// Set up mock expectations
	mockBackend.EXPECT().Name().Return("test-backend").AnyTimes()

	mockBackend.EXPECT().
		Dispatch(
			mockutil.Context(),
			jobID,
			"test_with_output",
			gomock.Any(),
		).
		Times(1).
		Return([]*dispatcher.TaskResult{
			{
				ExitCode: 0,
			},
		}, nil)

	_, err := dis.Dispatch(
		t.Context(),
		jobID,
		&workflow.LinkStandardTaskConfig{
			Execute: "test_with_output",
		},
		[]*dispatcher.TaskRequest{
			{
				StdoutFile: stdoutFile,
				StderrFile: stderrFile,
			},
		},
	)
	assert.NilError(t, err)
}
