package webui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"gotest.tools/v3/assert"
)

// testdata mimics assets (embed.FS) generated in spa.go.
var testdata = fstest.MapFS{
	"assets/index.html": &fstest.MapFile{
		Data: []byte("<!DOCTYPE html>"),
	},
	"assets/favicon.ico": &fstest.MapFile{
		Data: []byte("favicon"),
	},
}

func TestSPAHandler(t *testing.T) {
	t.Parallel()

	// Uncomment if you want to test against the real `assets`.
	// testdata := assets

	t.Run("Serves the index", func(t *testing.T) {
		t.Parallel()
		h := spaHandler(testdata)
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(rec, req)
		assertResponse(t, rec, http.StatusOK, []byte("DOCTYPE html"))
	})

	// https://router.vuejs.org/guide/essentials/history-mode#Example-Server-Configurations
	t.Run("Serves the index if the asset is not found", func(t *testing.T) {
		t.Parallel()
		h := spaHandler(testdata)
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/image.png", nil)
		h.ServeHTTP(rec, req)
		assertResponse(t, rec, http.StatusOK, []byte("DOCTYPE html"))
	})

	t.Run("Serves other assets", func(t *testing.T) {
		t.Parallel()
		h := spaHandler(testdata)
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/favicon.ico", nil)
		h.ServeHTTP(rec, req)
		assert.Equal(t, rec.Code, http.StatusOK)
		assert.Equal(t, rec.Header().Get("Content-Type"), "image/vnd.microsoft.icon")
	})
}

func assertResponse(t *testing.T, rec *httptest.ResponseRecorder, code int, contents []byte) {
	assert.Equal(t, rec.Code, code)
	assert.Assert(t, bytes.Contains(rec.Body.Bytes(), contents) == true)
}
