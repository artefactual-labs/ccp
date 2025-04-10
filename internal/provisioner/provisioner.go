package provisioner

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
)

// driver defines the interface for different provisioner implementations
// (e.g., local, podman). Implementations are responsible for managing the
// lifecycle of workers according to the provided configuration.
type driver interface {
	// Run starts the provisioning process, such as launching worker instances
	// or goroutines.
	//
	// This method should be non-blocking after the initial setup phase is
	// complete. The provided context `ctx` is primarily intended to allow
	// cancellation of the *startup* process. If startup is successful, Run
	// should return nil. If startup fails or is cancelled via `ctx`, an
	// appropriate error should be returned.
	//
	// The driver should manage the lifecycle of its workers independently of
	// this context once Run has returned successfully.
	Run(ctx context.Context) error

	// Stop initiates a graceful shutdown of all workers managed by the driver.
	//
	// It should signal all active workers to stop and wait for them to complete
	// their shutdown procedures. This method should block until either all
	// workers have stopped or the provided context `ctx` expires (indicating a
	// timeout). If shutdown completes successfully within the context's
	// deadline, it returns nil. If the shutdown times out or encounters other
	// errors, an appropriate error (potentially wrapping `ctx.Err()`) should be
	// returned.
	Stop(ctx context.Context) error
}

type Provisioner struct {
	logger logr.Logger
	driver driver
}

const (
	minWorkers = 1
	maxWorkers = 4
)

func New(logger logr.Logger, cfg Config) (*Provisioner, error) {
	if cfg.Count < minWorkers {
		cfg.Count = 1
	} else if cfg.Count > maxWorkers {
		return nil, fmt.Errorf("count must be between %d and %d", minWorkers, maxWorkers)
	}
	if cfg.Addr == nil {
		return nil, errors.New("addr cannot be nil")
	}
	if err := cfg.Type.Validate(); err != nil {
		return nil, err
	}

	logger = logger.WithName("worker").WithValues("type", cfg.Type)
	var driver driver
	var err error

	switch cfg.Type {
	case TypeNone:
		logger.Info("Using no-op provisioner, the application will not manage workers.")
		driver, err = newNoopDriver()
	case TypeLocal:
		logger.Info("Using local provisioner, workers will be run as goroutines.")
		driver = newLocalDriver(logger, cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create provisioner with driver %q: %v", cfg.Type, err)
	}

	p := &Provisioner{
		logger: logger,
		driver: driver,
	}

	return p, nil
}

func (p *Provisioner) Run(ctx context.Context) error {
	return p.driver.Run(ctx)
}

func (p *Provisioner) Close(ctx context.Context) error {
	return p.driver.Stop(ctx)
}
