package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/authn"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/storemock"
	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("Accepts API key", func(t *testing.T) {
		t.Parallel()

		store := storemock.NewMockStore(gomock.NewController(t))
		store.EXPECT().ValidateUserAPIKey(mockutil.Context(), "test", "test").Return(true, nil)

		auth := multiAuthenticate(authApiKey(logr.Discard(), store))
		handler := authn.NewMiddleware(auth).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.Header.Set("Authorization", "ApiKey test:test")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()

		assert.Equal(t, resp.StatusCode, http.StatusOK)
	})

	t.Run("Rejects invalid API key", func(t *testing.T) {
		t.Parallel()

		store := storemock.NewMockStore(gomock.NewController(t))
		store.EXPECT().ValidateUserAPIKey(mockutil.Context(), "test", "12345").Return(false, nil)

		auth := multiAuthenticate(authApiKey(logr.Discard(), store))
		handler := authn.NewMiddleware(auth).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.Header.Set("Authorization", "ApiKey test:12345")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()

		assert.Equal(t, resp.StatusCode, http.StatusUnauthorized)
	})
}
