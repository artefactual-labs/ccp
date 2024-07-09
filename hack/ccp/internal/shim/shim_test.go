package shim_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/ikawaha/httpcheck"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/shim"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/storemock"
)

func setUpShimServer(t *testing.T) *httpcheck.Checker {
	t.Helper()

	store := storemock.NewMockStore(gomock.NewController(t))
	store.EXPECT().ReadPipelineID(mockutil.Context()).Return(uuid.MustParse("9db764ac-84da-4c5f-a90d-872d4be54c3f"), nil).AnyTimes()

	srv := shim.NewServer(logr.Discard(), shim.Config{Addr: ":0"}, store)
	err := srv.Run()
	assert.NilError(t, err)

	t.Cleanup(func() { srv.Close(context.Background()) })

	return httpcheck.NewExternal(fmt.Sprintf("http://%s", srv.Addr()))
}

func readFile(t *testing.T, filename string) []byte {
	t.Helper()

	blob, err := os.ReadFile(filepath.Join("testdata", filename))
	assert.NilError(t, err)

	return blob
}

func TestShim(t *testing.T) {
	t.Parallel()

	t.Run("Includes ID and Version headers", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodGet, "/api/ingest/completed").
			Check().
			HasStatus(http.StatusOK).
			HasHeaders(map[string]string{
				"x-archivematica-version": "dev",
				"x-archivematica-id":      "9db764ac-84da-4c5f-a90d-872d4be54c3f",
			})
	})

	t.Run("Returns 404 if resource not found", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodGet, "/api/v0").
			Check().
			HasStatus(http.StatusNotFound)
	})

	t.Run("Returns 405 if method", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodGet, "/api/v2beta/validate/rights").
			Check().
			HasStatus(http.StatusMethodNotAllowed)
	})
}

func TestShimAdministrationFetchLevelsOfDescription(t *testing.T) {
	t.Parallel()

	c := setUpShimServer(t)

	c.Test(t, http.MethodGet, "/api/administration/dips/atom/fetch_levels").
		Check().
		HasStatus(http.StatusOK)
}

func TestShimValidateCreate(t *testing.T) {
	t.Parallel()

	t.Run("Validates Avalon CSV", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodPost, "/api/v2beta/validate/avalon").
			WithHeader("content-type", "text/csv; charset=utf-8").
			WithBody(readFile(t, "valid_avalon.csv")).
			Check().
			HasStatus(http.StatusOK).
			HasJSON(
				map[string]any{
					"valid": true,
				},
			)
	})

	t.Run("Returns error during validation of Avalon CSV", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodPost, "/api/v2beta/validate/avalon").
			WithHeader("content-type", "text/csv; charset=utf-8").
			WithBody(readFile(t, "invalid_avalon.csv")).
			Check().
			// TODO: should return JSON-encoded response with status code 400.
			HasStatus(http.StatusInternalServerError).
			HasString("manifest includes invalid metadata field: Bibliographic ID Lbl\n")
	})

	t.Run("Validates Rights CSV", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodPost, "/api/v2beta/validate/rights").
			WithHeader("content-type", "text/csv; charset=utf-8").
			WithBody(readFile(t, "valid_rights.csv")).
			Check().
			HasStatus(http.StatusOK).
			HasJSON(
				map[string]any{
					"valid": true,
				},
			)
	})

	t.Run("Returns error during validation of Rights CSV", func(t *testing.T) {
		t.Parallel()

		t.Skip("TODO")
	})

	t.Run("Fails is the validator is unknown", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodPost, "/api/v2beta/validate/unknown").
			Check().
			HasStatus(http.StatusNotFound).
			HasJSON(
				map[string]any{
					"error":   true,
					"message": "unknown validator, accepted values: avalon, rights",
				},
			)
	})

	t.Run("Fails if the content type is not the expected", func(t *testing.T) {
		t.Parallel()

		c := setUpShimServer(t)

		c.Test(t, http.MethodPost, "/api/v2beta/validate/avalon").
			WithBody(readFile(t, "valid_avalon.csv")).
			Check().
			HasStatus(http.StatusInternalServerError) // TODO: should be 400
	})
}
