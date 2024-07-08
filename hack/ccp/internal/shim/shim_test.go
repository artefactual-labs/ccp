package shim_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/shim"
)

func TestShim(t *testing.T) {
	t.Parallel()

	srv := shim.NewServer(logr.Discard(), shim.Config{Addr: ":0"})

	err := srv.Run()
	assert.NilError(t, err)
	t.Cleanup(func() { srv.Close(context.Background()) })

	// Returns 200.
	url := fmt.Sprintf("http://%s/api/administration/dips/atom/fetch_levels", srv.Addr())
	req, err := http.NewRequest(http.MethodGet, url, nil)
	assert.NilError(t, err)
	resp, err := http.DefaultClient.Do(req)
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	// Returns 404.
	url = fmt.Sprintf("http://%s/api/NOTFOUND", srv.Addr())
	req, err = http.NewRequest(http.MethodGet, url, nil)
	assert.NilError(t, err)
	resp, err = http.DefaultClient.Do(req)
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)
}
