package dispatcher_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/artefactual-labs/gearmin/gearmintest"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp/cmpopts"
	uuid "github.com/google/uuid"
	gearman_worker "github.com/mikespook/gearman-go/worker"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/store"
	"github.com/artefactual-labs/ccp/internal/store/storemock"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

func TestGearminDispatcher(t *testing.T) {
	t.Parallel()

	logger := logr.Discard()
	metrics := metrics.NewMetrics(nil)
	storeMock := storemock.NewMockStore(gomock.NewController(t))
	gearmin := gearmintest.Server(t, map[string]gearmintest.Handler{
		"func": funcHandler(t),
	})

	jobID := uuid.New()

	dis, err := dispatcher.New(logger, metrics, storeMock, gearmin, nil)
	assert.NilError(t, err)

	storeMock.EXPECT().CreateTasks(
		mockutil.Context(),
		mockutil.Eq(
			[]*store.Task{
				{
					JobID: jobID,
					Exec:  "func",
				},
			},
			cmpopts.IgnoreFields(store.Task{}, "ID", "CreatedAt"),
		)).Times(1)

	res, err := dis.Dispatch(
		t.Context(),
		jobID,
		&workflow.LinkStandardTaskConfig{
			Execute: "func",
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

func funcHandler(t *testing.T) gearmintest.Handler {
	return func(job gearman_worker.Job) ([]byte, error) {
		// Minimal structure to extract task UUIDs from the input JSON.
		var input struct {
			Tasks map[string]any `json:"tasks"`
		}
		err := json.Unmarshal(job.Data(), &input)
		assert.NilError(t, err)

		// Structure for the results for a single task.
		type taskResult struct {
			ExitCode   int       `json:"exitCode"`
			FinishedAt time.Time `json:"finishedTimestamp"`
			Stdout     string    `json:"stdout"`
			Stderr     string    `json:"stderr"`
		}

		// Build the results map.
		results := make(map[string]taskResult)
		now := time.Now()
		for taskUUID := range input.Tasks {
			// Validate if the key is actually a UUID, although for this mock
			// we just need the string keys from the input.
			if _, err := uuid.Parse(taskUUID); err != nil {
				continue
			}
			results[taskUUID] = taskResult{
				ExitCode:   0,
				FinishedAt: now,
				Stdout:     "mock stdout for " + taskUUID,
				Stderr:     "mock stderr for " + taskUUID,
			}
		}

		// Structure for the final response.
		response := struct {
			TaskResults map[string]taskResult `json:"task_results"`
		}{
			TaskResults: results,
		}

		// Marshal the response.
		respBytes, err := json.Marshal(response)
		assert.NilError(t, err)

		return respBytes, nil
	}
}
