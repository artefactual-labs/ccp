package controller

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/artefactual-labs/gearmin/gearmintest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/mikespook/gearman-go/worker"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestDirectoryClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Skip("Not implemented yet!")
}

func TestFilesClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Skip("Not implemented yet!")
}

func TestOutputClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Run("Updates chain choices with the output of the client script", func(t *testing.T) {
		t.Parallel()

		jobHandler := func(job worker.Job) ([]byte, error) {
			tasks := decodeTasks(t, job)
			task := tasks[0]

			return encodeTaskResults(t, map[uuid.UUID]*taskResult{
				task.ID: {
					ExitCode:   0,
					FinishedAt: time.Now(),
					Stdout: `{
						"default": {"description": "desc/1", "uri": "uri/1"},
						"5c732a52-6cdb-4b50-ac2e-ae10361b019a": {"description": "desc/2", "uri": "uri/2"}
					}`,
				},
			}), nil
		}

		job, store := createJobWithHandlers(t,
			"d026e5a4-96cf-4e4c-938d-a74b0d211da0", // Retrieve DIP Storage Locations.
			map[string]gearmintest.Handler{"getaipstoragelocations_v0.0": jobHandler},
		)
		createAutomatedProcessingConfig(t, job.pkg.path)

		store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(mockutil.Context(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		store.EXPECT().CreateTasks(mockutil.Context(), gomock.Any()).Return(nil).AnyTimes()

		_, err := job.exec(context.Background())
		assert.NilError(t, err)
		assert.DeepEqual(t,
			job.chain.choices,
			[]choice{
				{
					label:    "desc/1",
					value:    [2]string{"", "uri/1"},
					nextLink: uuid.Nil,
				},
				{
					label:    "desc/2",
					value:    [2]string{"", "uri/2"},
					nextLink: uuid.Nil,
				},
			},
			cmp.AllowUnexported(choice{}),
		)
	})
}

func decodeTasks(t *testing.T, job worker.Job) []*task {
	t.Helper()

	tasks := &tasks{}
	err := json.Unmarshal(job.Data(), tasks)
	assert.NilError(t, err)

	ret := make([]*task, 0, len(tasks.Tasks))
	for _, t := range tasks.Tasks {
		ret = append(ret, t)
	}

	return ret
}

func encodeTaskResults(t *testing.T, res map[uuid.UUID]*taskResult) []byte {
	t.Helper()

	taskResults := &taskResults{
		Results: res,
	}

	blob, err := json.Marshal(taskResults)
	assert.NilError(t, err)

	return blob
}
