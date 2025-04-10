package provisioner_test

import (
	"net"
	"testing"

	"github.com/go-logr/logr/testr"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/provisioner"
)

func createLocalProvisioner(t *testing.T, cfg provisioner.Config) *provisioner.Provisioner {
	t.Helper()

	logger := testr.NewWithInterface(t, testr.Options{})

	cfg.Type = provisioner.TypeLocal
	cfg.Addr = &net.TCPAddr{}

	driver, err := provisioner.New(logger, cfg)
	assert.NilError(t, err)

	return driver
}

func TestLocalDriver(t *testing.T) {
	t.Parallel()

	t.Run("Local driver runs and closes without error", func(t *testing.T) {
		t.Parallel()

		p := createLocalProvisioner(t, provisioner.Config{Count: 3})

		err := p.Run(t.Context())
		assert.NilError(t, err)

		err = p.Close(t.Context())
		assert.NilError(t, err)
	})
}
