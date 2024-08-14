package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/authn"
	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/store"
	"github.com/artefactual-labs/ccp/internal/store/storemock"
)

func TestAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("Accepts API key", func(t *testing.T) {
		t.Parallel()

		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := authn.GetInfo(r.Context()).(*store.User) // Rretrieve userinfo from context.
			assert.DeepEqual(t, user, &store.User{
				ID:       12345,
				Username: "test",
				Email:    "test@test.com",
				Active:   true,
			})
		})

		s := storemock.NewMockStore(gomock.NewController(t))
		s.EXPECT().ValidateUserAPIKey(mockutil.Context(), "test", "test").Return(&store.User{
			ID:       12345,
			Username: "test",
			Email:    "test@test.com",
			Active:   true,
		}, nil)

		auth := multiAuthenticate(authApiKey(logr.Discard(), s))
		handler = authn.NewMiddleware(auth).Wrap(handler)

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
		store.EXPECT().ValidateUserAPIKey(mockutil.Context(), "test", "12345").Return(nil, nil)

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
