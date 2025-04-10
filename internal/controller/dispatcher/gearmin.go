package dispatcher

import (
	context "context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	uuid "github.com/google/uuid"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/store"
)

const (
	gearminBackendID = "gearmin"

	// gearminBatchSize is the number of tasks we'll pack into each gearmin job.
	//
	// Chosen somewhat arbitrarily, but benchmarking with larger values (like
	// 512) didn't make much difference to throughput. Setting this too large
	// will use more memory; setting it too small will hurt throughput. So the
	// trick is to set it juuuust right.
	gearminBatchSize = 128
)

// gearminBackend submits tasks to MCPClient via gearmin.
type gearminBackend struct {
	logger  logr.Logger
	metrics *metrics.Metrics
	store   store.Store
	gearmin *gearmin.Server
}

var _ Backend = (*gearminBackend)(nil)

func newGearminBackend(logger logr.Logger, metrics *metrics.Metrics, store store.Store, gearmin *gearmin.Server) *gearminBackend {
	return &gearminBackend{
		logger:  logger.V(3),
		metrics: metrics,
		store:   store,
		gearmin: gearmin,
	}
}

func (b *gearminBackend) Name() string {
	return gearminBackendID
}

func (b *gearminBackend) Dispatch(ctx context.Context, jobID uuid.UUID, exec string, tasks []*Task) ([]*TaskResult, error) {
	if len(tasks) == 0 {
		return []*TaskResult{}, nil
	}

	var (
		batches     [][]*Task
		numBatches  int
		tasksByID   = make(map[uuid.UUID]*Task)
		resultsByID = make(map[uuid.UUID]*TaskResult)
		wg          sync.WaitGroup
		errMu       sync.Mutex
		anyErr      error
		resultsMu   sync.Mutex
	)

	// Prepare batches.
	for i := 0; i < len(tasks); i += gearminBatchSize {
		end := min(i+gearminBatchSize, len(tasks))
		batch := tasks[i:end]
		batches = append(batches, batch)
		numBatches++
	}

	// Save all tasks before submission.
	for _, batch := range batches {
		if err := b.saveTasks(ctx, jobID, exec, batch); err != nil {
			return nil, fmt.Errorf("save tasks: %v", err)
		}
		// Map tasks by UUID for fast lookup later (for output file writing).
		for _, t := range batch {
			tasksByID[t.ID] = t
		}
	}

	// We'll employ a goroutine for each batch.
	wg.Add(numBatches)

	for _, batch := range batches {
		data, batchTaskIDs, err := b.marshalBatch(batch)
		if err != nil {
			return nil, err
		}
		doneCh := make(chan *gearmin.JobUpdate, 1)

		// Lowercase function name for MCPClient.
		funcName := strings.ToLower(exec)

		// Submit to Gearmin.
		b.gearmin.Submit(&gearmin.JobRequest{
			ID:         uuid.NewString(),
			FuncName:   funcName,
			Data:       data,
			Background: false,
			Callback: func(update gearmin.JobUpdate) {
				doneCh <- &update
			},
		})

		// Wait and handle batch in goroutine.
		go func(tasksInBatch map[uuid.UUID]*Task, done <-chan *gearmin.JobUpdate) {
			defer wg.Done()
			select {
			case update := <-done:
				if err := b.handleJobUpdate(ctx, update, tasksInBatch, &resultsMu, resultsByID); err != nil {
					errMu.Lock()
					anyErr = errors.Join(anyErr, err)
					errMu.Unlock()
				}
			case <-ctx.Done():
				errMu.Lock()
				anyErr = errors.Join(anyErr, ctx.Err())
				errMu.Unlock()
			}
		}(batchTaskIDs, doneCh)
	}

	// Wait until all batches are done.
	wg.Wait()

	if anyErr != nil {
		return nil, anyErr
	}

	// Collate results as slice in order.
	taskResults := make([]*TaskResult, 0, len(tasks))
	for _, t := range tasks {
		if r, ok := resultsByID[t.ID]; ok {
			taskResults = append(taskResults, r)
		}
	}

	return taskResults, nil
}

// saveTasks persists all task records before submitting the batch as MCPClient
// then updates them.
func (g *gearminBackend) saveTasks(ctx context.Context, jobID uuid.UUID, exec string, batch []*Task) error {
	storeTasks := make([]*store.Task, 0, len(batch))
	for _, t := range batch {
		storeTasks = append(storeTasks, &store.Task{
			ID:        t.ID,
			CreatedAt: t.CreatedAt,
			Exec:      exec,
			Arguments: t.Request.Arguments,
			JobID:     jobID,
			FileID: uuid.NullUUID{
				UUID:  t.Request.FileID,
				Valid: t.Request.FileID != uuid.Nil,
			},
			Filename: t.Request.RelativeLocation,
		})
	}
	return g.store.CreateTasks(ctx, storeTasks)
}

// handleJobUpdate decodes the job result, updates result slice, and writes
// output to disk.
func (g *gearminBackend) handleJobUpdate(
	ctx context.Context,
	update *gearmin.JobUpdate,
	tasksInBatch map[uuid.UUID]*Task,
	resultsMu *sync.Mutex,
	results map[uuid.UUID]*TaskResult,
) error {
	// Only handle complete (success) events.
	if update == nil || update.Type != gearmin.JobUpdateTypeComplete {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Python client encodes result as a dict: {task_results: {uuid: {...}}}
	type batchResult struct {
		TaskResults map[uuid.UUID]*struct {
			ExitCode   int       `json:"exitCode"`
			FinishedAt time.Time `json:"finishedTimestamp"`
			Stdout     string    `json:"stdout"`
			Stderr     string    `json:"stderr"`
		} `json:"task_results"`
	}
	var res batchResult
	if err := json.Unmarshal(update.Data, &res); err != nil {
		return fmt.Errorf("unmarshal batch result: %w", err)
	}

	for taskID, tr := range res.TaskResults {
		// Write output if needed.
		task := tasksInBatch[taskID]
		if task != nil {
			_ = writeOutput(task.Request.StdoutFile, tr.Stdout)
			_ = writeOutput(task.Request.StderrFile, tr.Stderr)
		}
		result := &TaskResult{
			ExitCode:   tr.ExitCode,
			CreatedAt:  task.CreatedAt,
			FinishedAt: tr.FinishedAt,
		}
		resultsMu.Lock()
		results[taskID] = result
		resultsMu.Unlock()
	}
	return nil
}

// marshalBatch marshals the given batch as a Gearmin-compatible payload
// and returns the marshaled data and a map of task IDs to task pointers.
func (g *gearminBackend) marshalBatch(batch []*Task) ([]byte, map[uuid.UUID]*Task, error) {
	gearminPayload := make(map[uuid.UUID]*gearminTask, len(batch))
	originalTasksByID := make(map[uuid.UUID]*Task, len(batch))

	for _, t := range batch {
		gt := &gearminTask{
			ID:          t.ID,
			CreatedAt:   t.CreatedAt,
			Arguments:   t.Request.Arguments,
			WantsOutput: t.Request.captureOutput(),
		}
		gearminPayload[t.ID] = gt
		originalTasksByID[t.ID] = t
	}

	// Marshal the gearminPayload map within the expected structure:
	// { "tasks": { uuid: gearminTask, ... } }
	data, err := json.Marshal(struct {
		Tasks map[uuid.UUID]*gearminTask `json:"tasks"`
	}{Tasks: gearminPayload})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal tasks for gearmin: %w", err)
	}

	return data, originalTasksByID, nil
}

type gearminTask struct {
	ID        uuid.UUID `json:"task_uuid"`
	CreatedAt time.Time `json:"createdDate"`
	Arguments string    `json:"arguments"`

	// WantsOutput is used to request the worker to capture the output of the
	// task (aka MCPClient job) and send it back to us. This preference is
	// ignored if the worker has disabled client script output capturing.
	WantsOutput bool `json:"wants_output"`
}

func (t gearminTask) MarshalJSON() ([]byte, error) {
	// Python 3.10 or older can't parse the encoded output of time.Time, but it
	// is fixed in Python 3.11. We override the value here with a format that is
	// compatible.
	type alias gearminTask
	type out struct {
		alias
		CreatedAt string `json:"createdDate"`
	}
	createdAt := t.CreatedAt.Format("2006-01-02T15:04:05") +
		fmt.Sprintf(".%06d", t.CreatedAt.Nanosecond()/1000)[:7] +
		t.CreatedAt.Format("-07:00")
	return json.Marshal(out{
		alias:     alias(t),
		CreatedAt: createdAt,
	})
}
