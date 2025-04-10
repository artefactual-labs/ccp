package workhub

import (
	"errors"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

var (
	ErrWorkerNotFound          = errors.New("worker not found in registry")
	ErrWorkerAlreadyRegistered = errors.New("worker already registered")
)

// WorkerStatus represents the status of a worker in the registry.
type WorkerStatus string

const (
	// StatusIdle indicates the worker is registered and available to grab a
	// new batch.
	StatusIdle WorkerStatus = "idle"

	// StatusWorking indicates the worker has been assigned a batch and is
	// expected to be processing it.
	StatusWorking WorkerStatus = "working"
)

// WorkerInfo contains information about a worker.
type WorkerInfo struct {
	// Only used as a key in the map, here for clarity.
	WorkerID uuid.UUID
	// Status indicates the current status of the worker.
	Status WorkerStatus
	// LastSeen is the last time the worker sent a heartbeat signal.
	// This is used to determine if the worker is still active.
	LastSeen time.Time
}

type Registry struct {
	logger  logr.Logger
	workers map[uuid.UUID]*WorkerInfo
	mu      sync.RWMutex
}

func NewRegistry(logger logr.Logger) *Registry {
	return &Registry{
		logger:  logger,
		workers: make(map[uuid.UUID]*WorkerInfo),
	}
}

// RegisterOrUpdateWorker adds a new worker or updates an existing one.
// This should be called just before GrabJob returns.
func (r *Registry) RegisterOrUpdateWorker(workerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if worker, exists := r.workers[workerID]; exists {
		if worker.Status == StatusWorking {
			return ErrWorkerAlreadyRegistered
		}
		worker.Status = StatusIdle
		worker.LastSeen = now

		return nil
	}

	// Register a new worker instead.
	r.workers[workerID] = &WorkerInfo{
		WorkerID: workerID,
		Status:   StatusIdle,
		LastSeen: now,
	}

	return nil
}

// UpdateWorkerHeartbeat updates the LastSeen timestamp for a known worker.
// This should be called when a worker sends a heartbeat signal.
func (r *Registry) UpdateWorkerHeartbeat(workerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	worker, exists := r.workers[workerID]
	if !exists {
		return ErrWorkerNotFound
	}

	worker.LastSeen = time.Now()

	return nil
}

// MarkWorkerWorking sets the status of a known worker to Working.
// This should be called just before returning a job in GrabJobResponse.
func (r *Registry) MarkWorkerWorking(workerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	worker, exists := r.workers[workerID]
	if !exists {
		return ErrWorkerNotFound
	}

	worker.Status = StatusWorking
	worker.LastSeen = time.Now()

	return nil
}

// MarkWorkerIdle sets the status of a known worker to Idle.
// This should be called when a worker successfully reports job completion.
func (r *Registry) MarkWorkerIdle(workerID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	worker, exists := r.workers[workerID]
	if !exists {
		return ErrWorkerNotFound
	}

	worker.Status = StatusIdle
	worker.LastSeen = time.Now()

	return nil
}

// CleanUpExpiredWorkers finds workers whose LastSeen timestamp is older than
// the specified timeout duration and removes them from the registry. It returns
// the IDs of the workers that were removed.
//
// TODO: track batchID in WorkerInfo so we can return orphaned batches here
// allowing the caller (the hub loop) to locate the result channel and send an
// error to the original Enqueue caller.
func (r *Registry) CleanUpExpiredWorkers(timeout time.Duration) []uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()

	var removedIDs []uuid.UUID
	now := time.Now()

	for id, worker := range r.workers {
		if now.Sub(worker.LastSeen) > timeout {
			logger := r.logger.WithValues("worker_id", id, "last_seen", worker.LastSeen, "status", worker.Status)
			if worker.Status == StatusWorking {
				logger.Info("Expiring worker while it was in status Working - associated batch is now orphaned.")
			} else {
				logger.Info("Expiring worker due to inactivity.")
			}
			delete(r.workers, id)
			removedIDs = append(removedIDs, id)
		}
	}

	return removedIDs
}
