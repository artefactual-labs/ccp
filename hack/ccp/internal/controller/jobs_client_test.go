package controller

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/artefactual-labs/gearmin/gearmintest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/mikespook/gearman-go/worker"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestDirectoryClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Run("Runs a single task", func(t *testing.T) {
		t.Parallel()

		jobs := 0
		jobHandler := func(job worker.Job) ([]byte, error) {
			jobs++
			tasks := decodeTasks(t, job)
			task1 := tasks[0]
			assert.DeepEqual(t,
				task1,
				&task{
					ID:   task1.ID,
					Args: `"%SIPDirectory%" "%watchDirectoryPath%workFlowDecisions/compressionAIPDecisions/." "%SIPUUID%" "%sharedPath%"`,
				},
				cmp.AllowUnexported(task{}),
				cmpopts.IgnoreFields(task{}, "CreatedAt"),
			)
			return encodeTaskResults(t, map[uuid.UUID]*taskResult{
				task1.ID: {
					ExitCode:   0,
					FinishedAt: time.Now(),
					Stdout:     ``,
				},
			}), nil
		}

		job, store := createJobWithHandlers(t,
			"002716a1-ae29-4f36-98ab-0d97192669c4", // Move to compressionAIPDecisions directory.
			map[string]gearmintest.Handler{"movesip_v0.0": jobHandler},
		)
		createAutomatedProcessingConfig(t, job.pkg.path)

		store.EXPECT().CreateJob(mockutil.Context(), gomock.Any()).Return(nil).Times(1)
		store.EXPECT().UpdateJobStatus(mockutil.Context(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		store.EXPECT().CreateTasks(mockutil.Context(), gomock.Any()).Return(nil).AnyTimes()

		_, err := job.exec(context.Background())
		assert.ErrorIs(t, err, io.EOF) // End of chain.
		assert.Equal(t, jobs, 1)
	})
}

func TestFilesClientScriptJob(t *testing.T) {
	t.Parallel()

	t.Run("Runs multiple tasks", func(t *testing.T) {
		t.Parallel()

		jobs := 0
		jobHandler := func(job worker.Job) ([]byte, error) {
			jobs++
			return []byte(""), nil
		}

		job, _ := createJobWithHandlers(t,
			"0e41c244-6c3e-46b9-a554-65e66e5c9324", // Identify file format of attachments.
			map[string]gearmintest.Handler{"identifyfileformat_v0.0": jobHandler},
		)
		createAutomatedProcessingConfig(t, job.pkg.path)
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
