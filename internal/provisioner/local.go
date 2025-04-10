package provisioner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cleanhttp"

	workerv1connect "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1/workerv1beta1connect"
	"github.com/artefactual-labs/ccp/internal/worker"
)

// localDriver is a driver implementation for the local provisioner that manages
// worker goroutines.
//
// TODO: consider using a mechanism to monitor worker goroutines and potentially
// restart them if they exit unexpectedly.
type localDriver struct {
	logger  logr.Logger
	cfg     Config
	client  workerv1connect.WorkerServiceClient
	idGen   IdentifierGenerator          // Generator for stable worker IDs.
	mu      sync.Mutex                   // Mutex to protect shared state.
	running bool                         // Indicates if Run has been successfully called.
	cancel  context.CancelFunc           // Cancels the context passed to workers.
	wg      sync.WaitGroup               // Tracks active worker goroutines.
	workers map[uuid.UUID]*worker.Worker // Map worker ID to instance for management.
}

var _ driver = (*localDriver)(nil)

func newLocalDriver(logger logr.Logger, cfg Config) *localDriver {
	httpClient, baseURL := buildHTTPClient(cfg)
	client := workerv1connect.NewWorkerServiceClient(httpClient, baseURL)

	return &localDriver{
		logger:  logger,
		cfg:     cfg,
		client:  client,
		idGen:   NewUUIDGenerator(),
		workers: make(map[uuid.UUID]*worker.Worker),
	}
}

// Run starts the local provisioner, creating and starting the specified number
// of worker goroutines.
func (d *localDriver) Run(ctx context.Context) error { //nolint:contextcheck // runCtx is not cancelled by the input context.
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return errors.New("provisioner already running")
	}

	// Create a new context that will be passed to workers and cancelled by
	// the Stop method. We derive it from the background, not the input ctx,
	// because the latter only governs the startup of the provisioner, not its
	// entire lifetime.
	runCtx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.workers = make(map[uuid.UUID]*worker.Worker, d.cfg.Count)
	d.running = true
	d.mu.Unlock()

	d.logger.Info("Starting local workers...")
	var startErr error

	for i := range d.cfg.Count {
		// Check if the startup context itself was cancelled.
		select {
		case <-ctx.Done():
			startErr = fmt.Errorf("provisioner startup cancelled: %w", ctx.Err())
			goto cleanup
		default:
		}

		workerID := d.idGen.Yield(i)
		logger := d.logger.WithValues("index", i)

		w := worker.NewWorker(logger, d.client, worker.Config{WorkerID: workerID})

		// Store worker before starting its goroutine.
		d.mu.Lock()
		d.workers[workerID] = w
		d.mu.Unlock()

		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			logger.Info("Worker goroutine starting...")
			if err := w.Run(runCtx); err != nil {
				logger.Error(err, "Failed to initiate worker.")
				return
			}

			// Wait for the run context to be cancelled (triggered by d.Stop).
			<-runCtx.Done()

			stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer stopCancel()
			if err := w.Stop(stopCtx); err != nil && !errors.Is(err, context.Canceled) { //nolint:contextcheck // stopCtx is not cancelled by the input context.
				logger.Error(err, "Worker stopped with error.")
			} else {
				logger.Info("Worker stopped gracefully.")
			}

			d.mu.Lock()
			delete(d.workers, w.ID())
			d.mu.Unlock()
		}()
	}

	d.logger.Info("Local workers provisioned successfully.")
	return nil

cleanup:
	// This block is reached if ctx.Done() happened during the startup loop.
	d.logger.Error(startErr, "Cleaning up partially started workers due to startup cancellation.")

	// Need to stop the workers that *did* start. We don't call d.Stop directly
	// because d.running might not be fully consistent yet, and d.Stop has its
	// own locking. We'll mimic the core stop logic.
	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.mu.Unlock()

	// Use a separate, short timeout context for this cleanup wait.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()
	cleanupDone := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(cleanupDone)
	}()
	select {
	case <-cleanupDone:
	case <-cleanupCtx.Done():
		d.logger.Error(cleanupCtx.Err(), "Timed out waiting for partially started worker goroutines to exit.")
	}

	d.mu.Lock()
	d.running = false
	d.workers = make(map[uuid.UUID]*worker.Worker)
	d.mu.Unlock()

	return startErr
}

// Stop signals all running workers to shut down gracefully and waits for them
// to complete or until the provided context times out.
func (d *localDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.logger.Info("Stopping local workers...")
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.mu.Unlock()

	// Wait for all worker goroutines to finish their cleanup.
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	var waitErr error
	select {
	case <-done:
		d.logger.Info("All worker goroutines have exited.")
	case <-ctx.Done():
		waitErr = fmt.Errorf("timed out waiting for workers to stop: %w", ctx.Err())
		d.logger.Error(waitErr, "Timeout during stop.")

	}

	// We can now safely mark the provisioner as stopped.
	d.mu.Lock()
	d.running = false
	d.workers = make(map[uuid.UUID]*worker.Worker)
	d.mu.Unlock()

	if waitErr == nil {
		d.logger.Info("Local provisioner stopped successfully.")
	}

	return waitErr
}

// buildHTTPClient creates a plain-HTTP client using a pooled transport under
// the hood.
//
// We may want to use a different transport in the future, but for now
// this is a good starting point.
func buildHTTPClient(cfg Config) (*http.Client, string) {
	if cfg.Addr == nil {
		return nil, ""
	}

	// We could instead use a http2.Transport with AllowHTTP enabled to allow
	// HTTP/2 over cleartext (h2c).
	transport := cleanhttp.DefaultPooledTransport()

	// Override the dialer to use the configured address.
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, cfg.Addr.String())
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(float64(worker.DefaultGrabTimeout) * 1.25), // 25% more than DefaultGrabTimeout
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	baseURL := "http://" + cfg.Addr.String()

	return client, baseURL
}
