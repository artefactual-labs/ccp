package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

// batchSize is the number of files we'll pack into each MCPClient job.
//
// Chosen somewhat arbitrarily, but benchmarking with larger values (like 512)
// didn't make much difference to throughput. Setting this too large will use
// more memory; setting it too small will hurt throughput. So the trick is to
// set it juuuust right.
var batchSize = 128

// taskBackend submits tasks to MCPClient via Gearman.
//
// Tasks are batched into batchSize groups, serialized and sent to MCPClient.
// This adds some complexity but saves a lot of overhead.
//
// This is our first iteration and can be improved. A few ideas:
//   - Investigate overhead of sync.WaitGroup, do we have a better alternative?
//   - Introduce an object representing the batch, similar to GearmanTaskBatch.
//     It's an opportunity to hide `tasks` and `taskResults` with something more
//     succint or expressive.
//   - Deal with failed tasks, test it.
//   - Make the backend an application object for better resource management.
//   - Review injected dependencies and defined fields, some are unused?
type taskBackend struct {
	logger logr.Logger

	// job that generates the tasks.
	job *job

	// store is used to persist tasks.
	store store.Store

	// gearman is the job server we use to dispatch the tasks.
	gearman *gearmin.Server

	// Present in all client chain links: files, directories, output.
	config *workflow.LinkStandardTaskConfig

	// wg is used to wait until all batches are completed.
	wg sync.WaitGroup

	// tasks contains the entire set of tasks across tasks batches.
	tasks []*task

	// batch contains the set of batch for the current batch.
	batch []*task

	// count of batches used.
	count int

	// results contains the aggregated outcome of all batches.
	results *taskResults

	// mu is used to synchronize write access from handleJobUpdate.
	mu sync.Mutex
}

func newTaskBackend(logger logr.Logger, job *job, store store.Store, gearman *gearmin.Server, config *workflow.LinkStandardTaskConfig) *taskBackend {
	return &taskBackend{
		logger:  logger.V(3),
		job:     job,
		store:   store,
		gearman: gearman,
		batch:   make([]*task, 0, batchSize),
		results: &taskResults{
			Results: map[uuid.UUID]*taskResult{},
		},
		config: config,
	}
}

func (b *taskBackend) submit(ctx context.Context, rm replacementMapping, args string, wantsOutput bool, stdoutFilePath, stderrFilePath string) error {
	t := &task{
		ID:             uuid.New(),
		CreatedAt:      time.Now().UTC(),
		Args:           args,
		stdoutFilePath: stdoutFilePath,
		stderrFilePath: stderrFilePath,
		rm:             rm,
	}

	if wantsOutput || stdoutFilePath != "" || stderrFilePath != "" {
		t.WantsOutput = true
	}

	b.batch = append(b.batch, t)

	var err error
	if len(b.batch)%batchSize == 0 {
		err = b.sendBatch(ctx)
	}

	return err
}

func (b *taskBackend) sendBatch(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("send batch: %v", err)
		}
	}()

	ln := len(b.batch)
	if ln == 0 {
		return // Nothing to do
	}

	// Set the batch size to not exceed the predefined maximum. If the number
	// of tasks is fewer than this maximum, adjust the batch size to match the
	// total number of tasks. This prevents slicing beyond the bounds.
	size := batchSize
	if ln < batchSize {
		size = ln
	}

	// Store the last items in a new batch, update the original slice.
	batch := b.batch[ln-size:]
	b.batch = b.batch[:ln-size]

	// Keep track of all tasks in the job.
	b.mu.Lock()
	b.tasks = append(b.tasks, batch...)
	b.mu.Unlock()

	if err := b.saveTasks(ctx, batch); err != nil {
		return err
	}

	// The payload is shaped as a dictionary.
	payload := tasks{Tasks: make(map[uuid.UUID]*task, size)}
	for _, item := range batch {
		payload.Tasks[item.ID] = item
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal tasks: %v", err)
	}

	b.logger.Info("Submitting batch to MCPClient.", "script", b.config.Execute, "size", size)

	// Launch a goroutine to wait for this batch.
	done := make(chan *gearmin.JobUpdate, 1)
	b.wg.Add(1)
	go func() {
		defer func() {
			b.wg.Done()
		}()
		for {
			select {
			case update := <-done:
				b.handleJobUpdate(ctx, update)
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	b.gearman.Submit(
		&gearmin.JobRequest{
			ID:         uuid.NewString(),                  // Ensure uniqueness.
			FuncName:   strings.ToLower(b.config.Execute), // MCPClient lowercases the function name.
			Data:       data,
			Background: false,
			Callback: func(update gearmin.JobUpdate) {
				done <- &update
			},
		},
	)

	b.count++

	return nil
}

// saveTasks persists the tasks before they're used by MCPClient.
func (b *taskBackend) saveTasks(ctx context.Context, batch []*task) error {
	tt := make([]*store.Task, 0, len(batch))
	for _, item := range batch {
		task := &store.Task{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
			Exec:      b.config.Execute,
			Arguments: item.Args,
			JobID:     b.job.id,
		}

		if val, ok := item.rm["%fileUUID%"]; ok {
			if id, err := uuid.Parse(string(val)); err == nil {
				task.FileID.UUID = id
				task.FileID.Valid = true
			}
		}
		if val, ok := item.rm["%relativeLocation%"]; ok {
			if path, err := filepath.Abs(string(val)); err == nil {
				task.Filename = filepath.Base(path)
			}
		}

		tt = append(tt, task)
	}

	return b.store.CreateTasks(ctx, tt)
}

func (b *taskBackend) handleJobUpdate(ctx context.Context, update *gearmin.JobUpdate) {
	if err := ctx.Err(); err != nil {
		return
	}

	b.logger.Info("Received job update from worker.", "type", update.Type)

	var data []byte

	switch update.Type {
	// TODO: handle exceptions (with payload) and failures.
	case gearmin.JobUpdateTypeException:
		fallthrough
	case gearmin.JobUpdateTypeFail:
		return
	case gearmin.JobUpdateTypeComplete:
		data = update.Data
	}

	res := &taskResults{}
	if err := json.Unmarshal(data, res); err != nil {
		b.logger.V(3).Info("Failed to decode results of a batch.", "type", update.Type)
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, task := range b.tasks {
		id := task.ID
		if r, ok := res.Results[id]; ok {
			b.results.Results[id] = r
			_ = task.writeOutput(r.Stdout, r.Stderr)
		}
	}
}

func (b *taskBackend) wait(ctx context.Context) (*taskResults, error) {
	// Check if we have anything for this job that hasn't been submitted.
	if len(b.batch) > 0 {
		if err := b.sendBatch(ctx); err != nil {
			return nil, err
		}
	}

	if err := b.waitGroup(ctx); err != nil {
		return nil, err
	}

	var err error // TODO: capture errors.
	b.logger.Info("Completed all batches.", "batches", b.count, "tasks", len(b.results.Results), "err", err)

	return b.results, err
}

func (b *taskBackend) waitGroup(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(ch)
	}()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type tasks struct {
	Tasks map[uuid.UUID]*task `json:"tasks"`
}

func (t tasks) MarshalJSON() ([]byte, error) {
	if len(t.Tasks) == 0 {
		return nil, errors.New("map is empty")
	}
	type alias tasks
	return json.Marshal(&struct{ *alias }{alias: (*alias)(&t)})
}

type task struct {
	ID          uuid.UUID `json:"task_uuid"`
	CreatedAt   time.Time `json:"createdDate"`
	Args        string    `json:"arguments"`
	WantsOutput bool      `json:"wants_output"`

	rm             replacementMapping
	stdoutFilePath string
	stderrFilePath string

	// exitCode       *int
	// completedAt    time.Time
}

func (t task) MarshalJSON() ([]byte, error) {
	// Python 3.10 or older can't parse the encoded output of time.Time, but it
	// is fixed in Python 3.11. We override the value here with a format that is
	// compatible.
	type alias task
	type transformer struct {
		alias
		CreatedAt string `json:"createdDate"`
	}
	createdAt := t.CreatedAt.Format("2006-01-02T15:04:05") +
		fmt.Sprintf(".%06d", t.CreatedAt.Nanosecond()/1000)[:7] +
		t.CreatedAt.Format("-07:00")
	aliased := transformer{
		alias:     alias(t),
		CreatedAt: createdAt,
	}
	return json.Marshal(aliased)
}

// writeOutput writes the stdout/stderr we got from MCPClient out to files if
// necessary.
func (t *task) writeOutput(stdout, stderr string) (err error) {
	if t.stdoutFilePath != "" && stdout != "" {
		err = errors.Join(err, t.writeFile(t.stdoutFilePath, stdout))
	}
	if t.stderrFilePath != "" && stderr != "" {
		err = errors.Join(err, t.writeFile(t.stderrFilePath, stderr))
	}

	return err
}

func (t *task) writeFile(path, contents string) error {
	const mode = 0o750

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write([]byte(contents)); err != nil {
		return err
	}

	if err := os.Chmod(path, mode); err != nil {
		return err
	}

	return nil
}

type taskResults struct {
	Results map[uuid.UUID]*taskResult `json:"task_results"`
}

func (tr taskResults) First() *taskResult {
	var r *taskResult
	for _, tr := range tr.Results {
		r = tr
		break
	}
	return r
}

func (tr taskResults) ExitCode() int {
	var code int
	for _, task := range tr.Results {
		if task.ExitCode > 0 {
			code = task.ExitCode
		}
	}
	return code
}

type taskResult struct {
	ExitCode   int       `json:"exitCode"`
	FinishedAt time.Time `json:"finishedTimestamp"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
}
