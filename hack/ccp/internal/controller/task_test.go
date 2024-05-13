package controller

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/artefactual-labs/gearmin/gearmintest"
	"github.com/go-logr/logr/testr"
	"github.com/google/uuid"
	"github.com/mikespook/gearman-go/worker"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/storemock"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

// workerHandler is a fake worker function.
func workerHandler(t *testing.T, job worker.Job) ([]byte, error) {
	t.Helper()

	defer func() {
		if err := recover(); err != nil {
			t.Logf("testHandler panic recovered: %v", err)
		}
	}()

	// Decode request.
	tasks := &tasks{}
	err := json.Unmarshal(job.Data(), tasks)
	assert.NilError(t, err)

	// Encode response.
	ret := &taskResults{Results: map[uuid.UUID]*taskResult{}}
	for _, task := range tasks.Tasks {
		ret.Results[task.ID] = &taskResult{
			ExitCode:   0,
			FinishedAt: time.Now(),
			Stdout:     "stdout",
			Stderr:     "stderr",
		}
	}
	data, err := json.Marshal(ret)
	assert.NilError(t, err)

	return data, nil
}

func TestTaskBackend(t *testing.T) {
	t.Parallel()

	t.Skip("Needs to be fixed: https://github.com/artefactual-labs/gearmin/issues/3.")

	batchSize = 128
	fnName := "do"

	tmpDir := fs.NewDir(t, "ccp")

	var runs int
	ctx := context.Background()
	srv := gearmintest.Server(t, map[string]gearmintest.Handler{
		fnName: func(job worker.Job) ([]byte, error) {
			runs++
			return workerHandler(t, job)
		},
	})

	s := storemock.NewMockStore(gomock.NewController(t))
	s.EXPECT().CreateTasks(gomock.Any(), gomock.Cond(func(tt any) bool {
		tasks := tt.([]*store.Task)
		return len(tasks) <= batchSize // It should never exceed the batch size.
	})).AnyTimes()

	logger := testr.NewWithOptions(t, testr.Options{Verbosity: 10})
	backend := newTaskBackend(logger, &job{}, s, srv, &workflow.LinkStandardTaskConfig{
		Execute:    fnName,
		StdoutFile: tmpDir.Join("stdout.log"),
		StderrFile: tmpDir.Join("stderr.log"),
	})

	// Submit 1k jobs, i.e. 8 batches.
	for range 1000 {
		rm := replacementMapping{}
		backend.submit(ctx, rm, "args", true, tmpDir.Join("stdout.log"), tmpDir.Join("stderr.log"))
	}

	res, err := backend.wait(ctx)

	assert.NilError(t, err)
	assert.Equal(t, runs, 8)
	assert.DeepEqual(t, len(res.Results), 1000)

	assert.Assert(t, fs.Equal(tmpDir.Path(), fs.Expected(t,
		fs.WithFile("stdout.log", "", fs.MatchAnyFileContent, fs.WithMode(0o750)),
		fs.WithFile("stderr.log", "", fs.MatchAnyFileContent, fs.WithMode(0o750)),
	)))
}

func TestTasksEncoding(t *testing.T) {
	t.Parallel()

	encoded := `{"tasks":{"5ef281c2-692f-49a2-b8dd-36ab4e2beca5":{"task_uuid":"5ef281c2-692f-49a2-b8dd-36ab4e2beca5","arguments":"\"%sharedPath%\"","wants_output":true,"createdDate":"2024-04-12T05:40:20.123456+00:00"}}}`
	tasks := tasks{
		Tasks: map[uuid.UUID]*task{
			uuid.MustParse("5ef281c2-692f-49a2-b8dd-36ab4e2beca5"): {
				ID:          uuid.MustParse("5ef281c2-692f-49a2-b8dd-36ab4e2beca5"),
				CreatedAt:   time.Date(2024, time.April, 12, 5, 40, 20, 123456000, time.UTC),
				Args:        "\"%sharedPath%\"",
				WantsOutput: true,
			},
		},
	}

	blob, err := json.Marshal(&tasks)

	assert.NilError(t, err)
	assert.Equal(t, string(blob), encoded)
}

func TestTaskResultsDecoding(t *testing.T) {
	t.Parallel()

	encoded := `{
		"task_results": {
			"89991b27-6276-4d83-bf8c-f62e6c2f9587": {
				"exitCode": 0,
				"finishedTimestamp": "2024-04-12T05:40:20.123456+00:00",
				"stdout": "data",
				"stderr": "data"
			}
		}
	}`

	ts := taskResults{}
	err := json.Unmarshal([]byte(encoded), &ts)

	assert.NilError(t, err)
	assert.DeepEqual(t, ts, taskResults{
		Results: map[uuid.UUID]*taskResult{
			uuid.MustParse("89991b27-6276-4d83-bf8c-f62e6c2f9587"): {
				ExitCode:   0,
				FinishedAt: time.Date(2024, time.April, 12, 5, 40, 20, 123456000, time.UTC),
				Stdout:     "data",
				Stderr:     "data",
			},
		},
	})
}
