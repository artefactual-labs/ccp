package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/store"
	"github.com/artefactual-labs/ccp/internal/workhub"
)

const (
	workhubBackendID = "workhub"

	// workhubBatchSize is the number of tasks we'll pack into each workhub batch.
	// Using same size as gearmin for consistency.
	workhubBatchSize = 128
)

// workhubBackend submits tasks to the workhub.
type workhubBackend struct {
	logger  logr.Logger
	metrics *metrics.Metrics
	store   store.Store

	// submitter is the interface for submitting tasks to the workhub.
	submitter workhubSubmitter
}

var _ Backend = (*workhubBackend)(nil)

func newWorkhubBackend(logger logr.Logger, metrics *metrics.Metrics, store store.Store, submitter workhubSubmitter) *workhubBackend {
	return &workhubBackend{
		logger:    logger.V(3),
		metrics:   metrics,
		store:     store,
		submitter: submitter,
	}
}

func (b *workhubBackend) Name() string {
	return workhubBackendID
}

func (b *workhubBackend) Dispatch(ctx context.Context, jobID uuid.UUID, exec string, tasks []*Task) ([]*TaskResult, error) {
	if len(tasks) == 0 {
		return []*TaskResult{}, nil
	}

	// Save all tasks before submission.
	if err := b.saveTasks(ctx, jobID, exec, tasks); err != nil {
		return nil, fmt.Errorf("save tasks: %v", err)
	}

	// Prepare the tasks for submission to the workhub.
	pbTasks := make([]*workerv1.Task, 0, len(tasks))
	for _, task := range tasks {
		pbTasks = append(pbTasks, &workerv1.Task{
			Id:          task.ID.String(),
			Function:    exec,
			Arguments:   task.Request.Arguments,
			WantsOutput: task.Request.captureOutput(),
		})
	}

	pbTaskResults, err := b.submitter.Submit(ctx, exec, pbTasks)
	if err != nil {
		return nil, fmt.Errorf("submit: %v", err)
	}

	// Map the results back to the original tasks.
	results := make([]*TaskResult, len(tasks))
	for i, task := range tasks {
		taskID := task.ID.String()
		item, ok := pbTaskResults[taskID]
		if !ok {
			return nil, fmt.Errorf("task %s not found in results", taskID)
		}
		_ = writeOutput(task.Request.StdoutFile, item.Stdout)
		_ = writeOutput(task.Request.StderrFile, item.Stderr)

		// Save task completion
		finishedAt := item.FinishedAt.AsTime()
		if err := b.store.UpdateTaskCompletion(ctx, task.ID, int(item.ExitCode), string(item.Stdout), string(item.Stderr), finishedAt); err != nil {
			b.logger.Error(err, "failed to update task completion", "taskID", task.ID)
			// Continue processing other tasks rather than failing the whole batch
		}

		results[i] = &TaskResult{
			ExitCode:   int(item.ExitCode),
			CreatedAt:  task.CreatedAt,
			FinishedAt: finishedAt,
		}
	}

	return results, nil
}

// saveTasks persists all task records before submitting them to the workhub.
func (b *workhubBackend) saveTasks(ctx context.Context, jobID uuid.UUID, exec string, tasks []*Task) error {
	storeTasks := make([]*store.Task, 0, len(tasks))
	for _, t := range tasks {
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
	return b.store.CreateTasks(ctx, storeTasks)
}

// workhubSubmitter defines the interface for submitting work to the worker
// system and awaiting their results.
type workhubSubmitter interface {
	// Submit accepts a batch of tasks. It enqueues them for processing by
	// connected workers, and blocks until all tasks in the batch have reported
	// a result or the context is cancelled.
	Submit(ctx context.Context, fn string, tasks []*workerv1.Task) (map[string]*workerv1.TaskResult, error)
}

// workhubSubmitterImpl is a concrete implementation of the Submitter interface
// that uses the workhub to submit tasks. When a batch of tasks is submitted,
// the hub returns a PendingBatch that is used to wait for the results of the
// tasks in a blocking fashion.
type workhubSubmitterImpl struct {
	hub *workhub.Hub
}

var _ workhubSubmitter = (*workhubSubmitterImpl)(nil)

func NewWorkhubSubmitter(hub *workhub.Hub) workhubSubmitter {
	return &workhubSubmitterImpl{
		hub: hub,
	}
}

func (s *workhubSubmitterImpl) Submit(ctx context.Context, fn string, tasks []*workerv1.Task) (map[string]*workerv1.TaskResult, error) {
	if len(tasks) == 0 {
		return map[string]*workerv1.TaskResult{}, nil
	}

	var (
		batches    [][]*workerv1.Task
		numBatches int
		wg         sync.WaitGroup
		errMu      sync.Mutex
		anyErr     error
		resultsMu  sync.Mutex
		allResults = make(map[string]*workerv1.TaskResult)
	)

	// Prepare batches.
	for i := 0; i < len(tasks); i += workhubBatchSize {
		end := min(i+workhubBatchSize, len(tasks))
		batch := tasks[i:end]
		batches = append(batches, batch)
		numBatches++
	}

	// We'll employ a goroutine for each batch.
	wg.Add(numBatches)

	for _, batch := range batches {
		go func(batchTasks []*workerv1.Task) {
			defer wg.Done()

			// Check for context cancellation before starting work.
			select {
			case <-ctx.Done():
				errMu.Lock()
				anyErr = errors.Join(anyErr, ctx.Err())
				errMu.Unlock()
				return
			default:
			}

			pendingBatch, err := s.hub.Enqueue(ctx, fn, batchTasks)
			if err != nil {
				errMu.Lock()
				anyErr = errors.Join(anyErr, fmt.Errorf("enqueue batch: %v", err))
				errMu.Unlock()
				return
			}

			batchResults, err := pendingBatch.Wait(ctx)
			if err != nil {
				errMu.Lock()
				anyErr = errors.Join(anyErr, fmt.Errorf("wait for batch: %v", err))
				errMu.Unlock()
				return
			}

			// Merge results from this batch into the overall results.
			resultsMu.Lock()
			for taskID, result := range batchResults {
				allResults[taskID] = result
			}
			resultsMu.Unlock()
		}(batch)
	}

	// Wait until all batches are done.
	wg.Wait()

	if anyErr != nil {
		return nil, anyErr
	}

	return allResults, nil
}
