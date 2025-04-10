package dispatcher_test

import (
	"context"
	"testing"

	"github.com/artefactual-labs/gearmin/gearmintest"
	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/mikespook/gearman-go/worker"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher/dispatchermock"
	"github.com/artefactual-labs/ccp/internal/store/storemock"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

// dispatcherTestEnv is a test environment for a dispatcher with mocked backends.
type dispatcherTestEnv struct {
	dispatcher           *dispatcher.Dispatcher
	storeMock            *storemock.MockStore
	workhubSubmitterMock *dispatchermock.MockworkhubSubmitter
}

func setUpDispatcherTestEnv(tb testing.TB) *dispatcherTestEnv {
	tb.Helper()

	ctrl := gomock.NewController(tb)
	logger := testr.NewWithInterface(tb, testr.Options{Verbosity: 9})
	metrics := metrics.NewMetrics(nil)
	storeMock := storemock.NewMockStore(ctrl)
	gearman := gearmintest.Server(tb, map[string]gearmintest.Handler{
		"func": func(job worker.Job) ([]byte, error) {
			return nil, nil
		},
	})
	workhubSubmitter := dispatchermock.NewMockworkhubSubmitter(ctrl)

	d, err := dispatcher.New(logger, metrics, storeMock, gearman, workhubSubmitter)
	assert.Assert(tb, d != nil)
	assert.NilError(tb, err)

	return &dispatcherTestEnv{
		dispatcher:           d,
		storeMock:            storeMock,
		workhubSubmitterMock: workhubSubmitter,
	}
}

func TestDispatcher(t *testing.T) {
	t.Parallel()

	t.Run("Handles empty requests", func(t *testing.T) {
		t.Parallel()

		env := setUpDispatcherTestEnv(t)

		resp, err := env.dispatcher.Dispatch(t.Context(), uuid.New(), &workflow.LinkStandardTaskConfig{}, []*dispatcher.TaskRequest{})
		assert.Equal(t, len(resp), 0)
		assert.NilError(t, err)
	})

	t.Run("Routes to the workhub backend", func(t *testing.T) {
		t.Parallel()

		env := setUpDispatcherTestEnv(t)

		gomock.InOrder(
			env.storeMock.EXPECT().
				CreateTasks(mockutil.Context(), gomock.Any()).
				Times(1),
			env.workhubSubmitterMock.EXPECT().
				Submit(mockutil.Context(), "test_v0.0", gomock.Any()).
				Times(1).
				DoAndReturn(func(ctx context.Context, fn string, tasks []*workerv1.Task) (map[string]*workerv1.TaskResult, error) {
					assert.Equal(t, fn, "test_v0.0")
					assert.Equal(t, len(tasks), 1)
					return map[string]*workerv1.TaskResult{
						tasks[0].Id: {
							Id:         tasks[0].Id,
							ExitCode:   0,
							Stdout:     []byte{},
							Stderr:     []byte{},
							FinishedAt: timestamppb.Now(),
						},
					}, nil
				}),
			env.storeMock.EXPECT().
				UpdateTaskCompletion(mockutil.Context(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Times(1),
		)

		linkConfig := &workflow.LinkStandardTaskConfig{Execute: "test_v0.0"}
		req := []*dispatcher.TaskRequest{{}}
		resp, err := env.dispatcher.Dispatch(t.Context(), uuid.New(), linkConfig, req)

		assert.NilError(t, err)
		assert.DeepEqual(t, resp, []*dispatcher.TaskResult{
			{},
		}, cmpopts.IgnoreFields(dispatcher.TaskResult{}, "CreatedAt", "FinishedAt"))
	})
}
