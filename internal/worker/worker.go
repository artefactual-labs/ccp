package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
	workerv1connect "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1/workerv1beta1connect"
)

const (
	// DefaultHeartbeatInterval is set to 75% of the server's 1m interval. It is
	// slightly reduced as a preventive measure to avoid being marked as
	// "orphaned" if we accidentally exceed the 1m deadline.
	DefaultHeartbeatInterval = time.Minute * 3 / 4 // = 45s

	// DefaultGrabTimeout is the timeout for the Grab operation, with an extra
	// 25% added on top of the server’s blocking limit (1m) to cover potential
	// network or processing delays.
	DefaultGrabTimeout = time.Minute + time.Minute/4 // = 75s
)

type Config struct {
	WorkerID uuid.UUID
}

var DefaultProcessor processor = createProcessor()

// Worker encapsulates the logic for a single worker client instance.
type Worker struct {
	logger    logr.Logger
	cfg       Config
	client    workerv1connect.WorkerServiceClient
	processor processor

	// Configurable timeouts.
	heartbeatInterval time.Duration
	grabTimeout       time.Duration

	// Internal state for managing the run loop goroutine.
	wg     sync.WaitGroup
	cancel context.CancelFunc
	runErr error
	mu     sync.Mutex
}

// NewWorker creates a new worker instance.
func NewWorker(logger logr.Logger, client workerv1connect.WorkerServiceClient, cfg Config) *Worker {
	return &Worker{
		logger:            logger.WithValues("id", cfg.WorkerID),
		cfg:               cfg,
		client:            client,
		processor:         DefaultProcessor,
		heartbeatInterval: DefaultHeartbeatInterval,
		grabTimeout:       DefaultGrabTimeout,
	}
}

func (w *Worker) ID() uuid.UUID {
	return w.cfg.WorkerID
}

// Run starts the worker's main loop in a background goroutine and returns
// immediately. The provided context governs the *initiation* of shutdown;
// use Stop() to wait for graceful completion.
func (w *Worker) Run(ctx context.Context) error {
	w.mu.Lock()
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.mu.Unlock()

	w.wg.Add(1)
	go w.loop(runCtx)

	return nil
}

// loop runs the main worker loop, continuously grabbing batches and processing
// them until the context is cancelled or an error occurs.
func (w *Worker) loop(ctx context.Context) {
	defer w.wg.Done()

	// loopErr is used to capture any error that occurs in the loop.
	var loopErr error
	defer func() {
		w.mu.Lock()
		w.runErr = loopErr
		w.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			loopErr = ctx.Err()
			return
		default:
			batch, err := w.grabBatch(ctx)

			// Exit loop during shutdown.
			if errors.Is(err, context.Canceled) {
				loopErr = err
				return
			}

			// Simple retry delay. TODO: implement more robust retry logic.
			if err != nil {
				w.logger.Info("Failed to grab batch, retrying...", "code", connect.CodeOf(err), "err", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Continue the loop if no batch was received.
			// This can happen if the server has no work available.
			if batch == nil || len(batch.Tasks) == 0 {
				continue
			}

			// Block until the context is cancelled or the batch is processed.
			w.handleBatch(ctx, batch)
		}
	}
}

func (w *Worker) handleBatch(ctx context.Context, batch *workerv1.GrabBatchResponse) {
	// Start heartbeating while processing this batch.
	hbCtx, hbCancel := context.WithCancel(ctx)
	var hbWg sync.WaitGroup
	hbWg.Add(1)
	go func() {
		defer hbWg.Done()
		w.runHeartbeatLoop(hbCtx)
	}()

	results := w.processBatch(ctx, batch)

	// Wait until heartbeating stops.
	hbCancel()
	hbWg.Wait()

	err := w.completeBatch(ctx, batch.Id, results)
	if err != nil {
		w.logger.Error(err, "Failed to complete batch.", "batch-id", batch.Id)
		return
	}

	w.logger.Info("Batch completed.", "batch-id", batch.Id)
}

// runHeartbeatLoop sends heartbeats periodically until the context is cancelled.
func (w *Worker) runHeartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	// We use a background context for the heartbeat itself so it's not
	// cancelled prematurely if the main processing finishes just as a heartbeat
	// is sent.
	newCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 5*time.Second)
	}

	// Send initial heartbeat immediately.
	hbCtx, hbCancel := newCtx()
	if err := w.heartbeat(hbCtx); err != nil { //nolint:contextcheck // hbCtx is not cancelled by the input context.
		w.logger.Error(err, "Failed to send initial heartbeat.")
	}
	hbCancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hbCtx, hbCancel := newCtx()
			if err := w.heartbeat(hbCtx); err != nil { //nolint:contextcheck // hbCtx is not cancelled by the input context.
				w.logger.Error(err, "Failed to send heartbeat during batch processing.")
			}
			hbCancel()
		}
	}
}

// processBatch processes the tasks in the batch.
func (w *Worker) processBatch(ctx context.Context, batch *workerv1.GrabBatchResponse) []*workerv1.TaskResult {
	results, err := w.processor.Do(ctx, batch.Function, batch.Tasks)
	if err != nil {
		w.logger.Error(err, "Failed to process batch.", "batchID", batch.Id)
	}

	return results
}

// grabBatch sends a request to the server to grab a batch for processing.
func (w *Worker) grabBatch(ctx context.Context) (*workerv1.GrabBatchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, w.grabTimeout)
	defer cancel()

	req := connect.NewRequest(&workerv1.GrabBatchRequest{
		WorkerId: w.cfg.WorkerID.String(),
	})

	resp, err := w.client.GrabBatch(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

// completeBatch sends the results of a completed batch back to the server.
func (w *Worker) completeBatch(ctx context.Context, batchID string, results []*workerv1.TaskResult) error {
	req := connect.NewRequest(&workerv1.CompleteBatchRequest{
		WorkerId: w.cfg.WorkerID.String(),
		BatchId:  batchID,
		Results:  results,
	})

	// Use a shorter timeout for completing the batch.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if _, err := w.client.CompleteBatch(ctx, req); err != nil {
		connectErr := &connect.Error{}
		isConnectErr := errors.As(err, &connectErr)

		if errors.Is(err, context.Canceled) || (isConnectErr && connectErr.Code() == connect.CodeCanceled) {
			if ctx.Err() == context.Canceled {
				return context.Canceled // Main context cancelled.
			}
			return fmt.Errorf("CompleteBatch call cancelled: %w", err)
		}

		if errors.Is(err, context.DeadlineExceeded) || (isConnectErr && connectErr.Code() == connect.CodeDeadlineExceeded) {
			return fmt.Errorf("CompleteBatch call timed out: %w", err)
		}

		return fmt.Errorf("CompleteBatch call failed: %w", err)
	}

	return nil
}

// heartbeat sends a heartbeat message to the server.
func (w *Worker) heartbeat(ctx context.Context) error {
	req := connect.NewRequest(&workerv1.HeartbeatRequest{
		Id: w.cfg.WorkerID.String(),
	})

	if _, err := w.client.Heartbeat(ctx, req); err != nil {
		connectErr := &connect.Error{}
		isConnectErr := errors.As(err, &connectErr)

		if errors.Is(err, context.Canceled) || (isConnectErr && connectErr.Code() == connect.CodeCanceled) {
			return context.Canceled // Propagate cancellation.
		}

		if errors.Is(err, context.DeadlineExceeded) || (isConnectErr && connectErr.Code() == connect.CodeDeadlineExceeded) {
			return fmt.Errorf("heartbeat call timed out: %w", err)
		}

		return fmt.Errorf("heartbeat call failed: %w", err)
	}

	return nil
}

// Stop signals the worker to shut down gracefully and blocks until the run loop
// has completed or the provided context expires. It returns any error
// encountered during the run loop's execution or if the stop context expires.
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if w.cancel == nil {
		w.mu.Unlock()
		return nil // Not running, nothing to do.
	}
	w.cancel()     // Signal the loop to stop.
	w.cancel = nil // Mark as not running.
	w.mu.Unlock()

	// Channel to signal when wg.Wait() is done.
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.wg.Wait() // Wait for the loop goroutine to finish.
	}()

	select {
	case <-done:
		// Loop finished.
	case <-ctx.Done():
		w.mu.Lock()
		defer w.mu.Unlock()
		return fmt.Errorf("stop timed out: %w (run loop error: %v)", ctx.Err(), w.runErr)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.runErr // Return any error from the run loop.
}
