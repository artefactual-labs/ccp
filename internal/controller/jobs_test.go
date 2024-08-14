package controller

import (
	"context"
	"testing"

	"github.com/artefactual-labs/gearmin/gearmintest"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/mikespook/gearman-go/worker"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/store/enums"
	"github.com/artefactual-labs/ccp/internal/store/storemock"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

func createJob(t *testing.T, linkID string) (*job, *storemock.MockStore) {
	t.Helper()

	return createJobWithHandlers(t, linkID, nil)
}

func createJobWithHandlers(t *testing.T, linkID string, handlers map[string]gearmintest.Handler) (*job, *storemock.MockStore) {
	t.Helper()

	tmpDir := fs.NewDir(t, "ccp", fs.WithDir("sharedDir/tmp/pkg"))

	if handlers == nil {
		handlers = map[string]gearmintest.Handler{
			"hello": func(job worker.Job) ([]byte, error) {
				return []byte("hi!"), nil
			},
		}
	}
	gearmin := gearmintest.Server(t, handlers)
	wf, _ := workflow.Default()
	ln := wf.Links[uuid.MustParse(linkID)]
	store := storemock.NewMockStore(gomock.NewController(t))
	chain := newChain(nil)

	pkg := newPackage(logr.Discard(), store, tmpDir.Join("sharedDir"))
	pkg.id = uuid.New()
	pkg.unit = &noUnit{}
	pkg.path = tmpDir.Join("sharedDir/tmp/pkg")

	job, err := newJob(logr.Discard(), metrics.NewMetrics(nil), chain, pkg, gearmin, ln, wf)
	assert.NilError(t, err)

	return job, store
}

type noUnit struct{}

func (u *noUnit) hydrate(ctx context.Context, path, watchedDir string) error {
	return nil
}

func (u *noUnit) reload(ctx context.Context) error {
	return nil
}

func (u *noUnit) replacements(filterSubdirPath string) replacementMapping {
	return nil
}

func (u *noUnit) replacementPath() string {
	return ""
}

func (u *noUnit) packageType() enums.PackageType {
	return enums.PackageTypeTransfer
}

func (u *noUnit) jobUnitType() string {
	return ""
}
