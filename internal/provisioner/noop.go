package provisioner

import "context"

// noopDriver is a provisioner driver that does nothing.
type noopDriver struct{}

var _ driver = (*noopDriver)(nil)

func newNoopDriver() (*noopDriver, error) {
	return &noopDriver{}, nil
}

func (d *noopDriver) Run(ctx context.Context) error {
	return nil
}

func (d *noopDriver) Stop(ctx context.Context) error {
	return nil
}
