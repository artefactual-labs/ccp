// Package worker provides different mechanisms to run the processing worker.
package worker

import (
	"fmt"

	"github.com/artefactual-labs/ccp/internal/worker/driver"
	"github.com/artefactual-labs/ccp/internal/worker/runc"
)

type Pool struct {
	d driver.Driver
}

func New() (*Pool, error) {
	d, err := loadDriver()
	if err != nil {
		return nil, fmt.Errorf("unable to load driver: %v", err)
	}

	return &Pool{d}, nil
}

func loadDriver() (driver.Driver, error) {
	d, err := runc.Load()
	if err == driver.ErrNotAvailable {
		return &noop{}, nil
	} else if err != nil {
		return nil, err
	}

	return d, nil
}

func (p *Pool) Start() {
	p.d.Info()
}

func (p *Pool) Close() error {
	return nil
}
