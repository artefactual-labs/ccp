package webui

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
)

// reportPanic is middleware for catching panics and reporting them.
func reportPanic(logger logr.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					w.WriteHeader(http.StatusInternalServerError)
					if r == http.ErrAbortHandler {
						panic(r)
					}
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					logger.Error(err, "Panic recovered")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
