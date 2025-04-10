package workhub

import (
	"context"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
	workerv1connect "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1/workerv1beta1connect"
)

const (
	// DefaultWorkerExpirationTimeout defines how long a worker can be silent
	// before being considered expired. Should be longer than heartbeat
	// interval.
	DefaultWorkerExpirationTimeout = time.Minute

	// DefaultWorkerCleanupInterval defines how often the hub checks for
	// expired workers.
	DefaultWorkerCleanupInterval = time.Minute

	// DefaultGrabBatchPollTimeout defines how long the server holds a GrabBatch
	// request if no batch is immediately available.
	DefaultGrabBatchPollTimeout = time.Minute
)

// Hub implements the Worker API.
type Hub struct {
	logger logr.Logger

	// Registry to manage worker states.
	reg *Registry

	// Buffered channel for pending batches. Size depends on expected backlog.
	batchQueue chan *batch

	// Signal channel for new batches, we'll use a buffered channel (size 1)
	// for notification to avoid blocking the notifier.
	batchNotifier chan struct{}

	// Map to store result channels for pending batches.
	pendingBatchesByID   map[uuid.UUID]chan BatchResult
	pendingBatchesByIDMu sync.Mutex

	// Management of the lifecycle of the internal loop.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration for worker expiration.
	workerExpirationTimeout time.Duration
	workerCleanupInterval   time.Duration

	// Configuration for batch polling timeout.
	grabBatchPollTimeout time.Duration
}

var _ workerv1connect.WorkerServiceHandler = (*Hub)(nil)

func NewHub(logger logr.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	h := &Hub{
		logger:                  logger,
		reg:                     NewRegistry(logger.WithName("registry")),
		batchQueue:              make(chan *batch, 100),
		batchNotifier:           make(chan struct{}, 1),
		pendingBatchesByID:      make(map[uuid.UUID]chan BatchResult),
		pendingBatchesByIDMu:    sync.Mutex{},
		ctx:                     ctx,
		cancel:                  cancel,
		workerExpirationTimeout: DefaultWorkerExpirationTimeout,
		workerCleanupInterval:   DefaultWorkerCleanupInterval,
		grabBatchPollTimeout:    DefaultGrabBatchPollTimeout,
	}

	h.wg.Add(1)
	go h.loop()

	return h
}

func (h *Hub) loop() {
	defer h.wg.Done()
	h.logger.Info("Hub internal loop started.")

	ticker := time.NewTicker(h.workerCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			_ = h.reg.CleanUpExpiredWorkers(h.workerExpirationTimeout)
		}
	}
}

func (h *Hub) GrabBatch(ctx context.Context, req *connect.Request[workerv1.GrabBatchRequest]) (*connect.Response[workerv1.GrabBatchResponse], error) {
	workerID := uuid.MustParse(req.Msg.WorkerId)

	if err := h.reg.RegisterOrUpdateWorker(workerID); err != nil {
		h.logger.Error(err, "Failed to register worker.", "id", workerID)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	timeoutTimer := time.NewTimer(h.grabBatchPollTimeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case batch, ok := <-h.batchQueue:
			if !ok {
				// Hub must be shutting down because the channel is closed.
				return nil, connect.NewError(connect.CodeUnavailable, errors.New("server shutting down or queue closed"))
			}

			if err := h.reg.MarkWorkerWorking(workerID); err != nil {
				// Try to put the batch back non-blockingly.
				select {
				case h.batchQueue <- batch:
					h.logger.Info("Successfully requeued batch", "batchID", batch.id)
					h.notifyWaiters() // Notify others there's work again
				default:
					// This is bad - failed to mark worker AND failed to requeue. Batch is lost.
					h.logger.Error(errors.New("failed to requeue batch - queue might be full"), "Batch lost", "batchID", batch.id)
					// Continue the loop to potentially grab another batch or timeout,
					// but log the loss of the current batch.
					// Alternatively, return an internal error immediately.
					// return nil, connect.NewError(connect.CodeInternal, errors.New("server error processing batch assignment"))
				}

				// Whether requeue succeeded or failed, this attempt to grab failed for the worker.
				// Continue the loop to wait for another chance or timeout.
				continue // Go back to the start of the outer select
			}

			return connect.NewResponse(&workerv1.GrabBatchResponse{
				Id:       batch.id.String(),
				Function: batch.fn,
				Tasks:    batch.tasks,
			}), nil

		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeCanceled, ctx.Err())

		case <-timeoutTimer.C:
			resp := connect.NewResponse(&workerv1.GrabBatchResponse{})
			return resp, nil

		case <-h.batchNotifier:
			continue // Loop back to the top to attempt dequeue again.
		}
	}
}

func (h *Hub) CompleteBatch(ctx context.Context, req *connect.Request[workerv1.CompleteBatchRequest]) (*connect.Response[workerv1.CompleteBatchResponse], error) {
	workerID := uuid.MustParse(req.Msg.WorkerId)
	batchID := uuid.MustParse(req.Msg.BatchId)
	logger := h.logger.WithValues("worker-id", workerID, "batch-id", batchID)

	// Remove the result channel.
	h.pendingBatchesByIDMu.Lock()
	resultChan, ok := h.pendingBatchesByID[batchID]
	if ok {
		delete(h.pendingBatchesByID, batchID)
	}
	h.pendingBatchesByIDMu.Unlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("batch ID not found, may have already completed or timed out"))
	}

	// Prepare and send the results map.
	results := map[string]*workerv1.TaskResult{}
	for _, res := range req.Msg.Results {
		results[res.Id] = res
	}
	select {
	case resultChan <- results:
		logger.Info("Results sent successfully for batch.")
	default:
		logger.Info("Result channel receiver for batch was gone (likely timed out).")
	}

	// Close the channel regardless of whether the send succeeded. This signals
	// completion definitively to any potential waiters and cleans up.
	close(resultChan)

	if err := h.reg.MarkWorkerIdle(workerID); errors.Is(err, ErrWorkerNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, ErrWorkerNotFound)
	} else if err != nil {
		h.logger.Error(err, "Failed to mark worker as idle.")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workerv1.CompleteBatchResponse{}), nil
}

func (h *Hub) Heartbeat(ctx context.Context, req *connect.Request[workerv1.HeartbeatRequest]) (*connect.Response[workerv1.HeartbeatResponse], error) {
	workerID := uuid.MustParse(req.Msg.Id)
	h.logger.Info("Received heartbeat.", "worker-id", workerID)

	if err := h.reg.UpdateWorkerHeartbeat(workerID); errors.Is(err, ErrWorkerNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, ErrWorkerNotFound)
	} else if err != nil {
		h.logger.Error(err, "Failed to update worker heartbeat.", "worker-id", workerID)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workerv1.HeartbeatResponse{}), nil
}

func (h *Hub) Close() {
	h.cancel()
	h.wg.Wait()
	close(h.batchQueue)
	close(h.batchNotifier)
}

type batch struct {
	id    uuid.UUID        // Identifier for the batch.
	fn    string           // Function name to be executed.
	tasks []*workerv1.Task // Tasks to be processed in the batch.
}

// Enqueue batch for processing.
func (h *Hub) Enqueue(ctx context.Context, fn string, tasks []*workerv1.Task) (*PendingBatch, error) {
	b := &batch{id: uuid.New(), fn: fn, tasks: tasks}

	// The channel is buffered to prevent CompleteBatch from blocking if Wait
	// has not been called yet.
	resultChan := make(chan BatchResult, 1)

	h.pendingBatchesByIDMu.Lock()
	h.pendingBatchesByID[b.id] = resultChan
	h.pendingBatchesByIDMu.Unlock()

	select {
	case h.batchQueue <- b:
		h.notifyWaiters()
		return &PendingBatch{ID: b.id, resultChan: resultChan, hub: h}, nil
	case <-ctx.Done():
		h.pendingBatchesByIDMu.Lock()
		delete(h.pendingBatchesByID, b.id)
		h.pendingBatchesByIDMu.Unlock()
		return nil, ctx.Err()
	default:
		h.pendingBatchesByIDMu.Lock()
		delete(h.pendingBatchesByID, b.id)
		h.pendingBatchesByIDMu.Unlock()
		return nil, errors.New("batch queue is full")
	}
}

func (h *Hub) notifyWaiters() {
	select {
	case h.batchNotifier <- struct{}{}:
	default:
		h.logger.V(1).Info("Notification signal channel full or no waiters.")
	}
}

// Define the result type expected by PendingBatch.Wait
type BatchResult map[string]*workerv1.TaskResult

type PendingBatch struct {
	ID         uuid.UUID
	resultChan <-chan BatchResult // Receive-only channel for results
	hub        *Hub               // Reference back to the hub to allow cleanup
}

func (pb *PendingBatch) Wait(ctx context.Context) (BatchResult, error) {
	select {
	case <-ctx.Done():
		// Context cancelled or timed out before results arrived.
		// We need to clean up the entry in the hub's map.
		pb.hub.pendingBatchesByIDMu.Lock()
		delete(pb.hub.pendingBatchesByID, pb.ID)
		pb.hub.pendingBatchesByIDMu.Unlock()
		return nil, ctx.Err()
	case results, ok := <-pb.resultChan:
		if !ok {
			// Channel was closed without sending data. This might indicate an error
			// during processing in CompleteBatch before results could be sent,
			// or potentially a race condition during cleanup.
			return nil, errors.New("result channel closed unexpectedly")
		}
		// Results received successfully. The map entry was already removed by CompleteBatch.
		return results, nil
	}
}
