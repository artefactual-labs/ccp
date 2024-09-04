package runc

import (
	"fmt"

	"github.com/artefactual-labs/ccp/internal/worker/driver"
)

type pool struct{}

func Load() (*pool, error) {
	if !enabled {
		return nil, driver.ErrNotAvailable
	}

	return &pool{}, nil
}

func (p *pool) Info() {
	entries, err := assets.ReadDir("assets")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		fmt.Println(entry.Name())
	}
}
