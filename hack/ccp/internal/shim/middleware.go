package shim

import (
	"context"
	"net/http"

	"github.com/artefactual/archivematica/hack/ccp/internal/shim/gen"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
)

const (
	archivematicaVersion = "dev"
)

func infoMiddleware(store store.Store) func(next gen.StrictHandlerFunc, _ string) gen.StrictHandlerFunc {
	pipelineID := ""

	return func(next gen.StrictHandlerFunc, _ string) gen.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
			if pipelineID == "" {
				if id, err := store.ReadPipelineID(ctx); err != nil {
					return nil, err
				} else {
					pipelineID = id.String()
				}
			}
			w.Header().Set("X-Archivematica-Version", archivematicaVersion)
			w.Header().Set("x-Archivematica-ID", pipelineID)
			return next(ctx, w, r, request)
		}
	}
}

type contextKey string

const (
	requestContextKey contextKey = "requestObject"
)

func contextMiddleware(next gen.StrictHandlerFunc, _ string) gen.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
		ctx = context.WithValue(ctx, requestContextKey, r)
		return next(ctx, w, r, request)
	}
}

func requestFromContext(ctx context.Context) *http.Request {
	if req, ok := ctx.Value(requestContextKey).(*http.Request); ok {
		return req
	}

	return nil
}
