package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed assets/*
var assets embed.FS

func spaHandler(assetsFS fs.FS) http.HandlerFunc {
	assetsFS, _ = fs.Sub(assetsFS, "assets")
	fileServer := http.FileServer(http.FS(assetsFS))

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		if _, err := fs.Stat(assetsFS, path); os.IsNotExist(err) {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	}
}
