package provisioner

import (
	"flag"
	"fmt"
	"net"
)

// Type represents the kind of provisioner driver to use.
type Type string

var _ flag.Value = (*Type)(nil)

const (
	// TypeNone indicates that no provisioner should be used, i.e. workers are
	// managed externally.
	TypeNone Type = ""
	// TypeLocal indicates a provisioner that runs workers as goroutines.
	TypeLocal Type = "local"
)

// String implements the flag.Value interface.
func (t *Type) String() string {
	return string(*t)
}

// Set implements the flag.Value interface.
func (t *Type) Set(value string) error {
	tt := Type(value)
	if err := tt.Validate(); err != nil {
		return err
	}
	*t = tt
	return nil
}

// Validate checks if the Type is one of the known, supported values.
func (t Type) Validate() error {
	switch t {
	case TypeNone, TypeLocal:
		return nil
	default:
		return fmt.Errorf("unsupported provisioner type: %q", t)
	}
}

type Config struct {
	// Type specifies the provisioner driver implementation to use.
	// If empty (TypeNone), a no-op provisioner is used.
	Type Type

	// Count specifies the number of workers to start.
	Count int

	// Addr provides the workhub network endpoint address. Not a configurable
	// option, this is populated by the server.
	Addr net.Addr
}
