package driver

import "errors"

var ErrNotAvailable = errors.New("driver not available")

type Driver interface {
	Info()
}
