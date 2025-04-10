package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher/dispatchermock"
	"github.com/artefactual-labs/ccp/internal/store/enums"
	"github.com/artefactual-labs/ccp/internal/store/storemock"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

type testJob struct {
	job *job

	store      *storemock.MockStore
	backend    *dispatchermock.MockBackend
	dispatcher *dispatcher.Dispatcher
}

// createJob creates a new job for testing purposes.
func createJob(t *testing.T, linkID string) *testJob {
	t.Helper()

	logger := logr.Discard()
	metrics := metrics.NewMetrics(nil)
	ctrl := gomock.NewController(t)
	backend := dispatchermock.NewMockBackend(ctrl)
	store := storemock.NewMockStore(ctrl)
	dispatcher := dispatcher.NewWithBackend(logger, metrics, store, backend)

	wf, _ := workflow.Default()
	ln := wf.Links[uuid.MustParse(linkID)]
	chain := newChain(nil)

	tmpDir := fs.NewDir(t, "ccp", fs.WithDir("sharedDir/tmp/pkg"))
	pkg := newPackage(logr.Discard(), store, tmpDir.Join("sharedDir"))
	pkg.id = uuid.New()
	pkg.unit = &noUnit{}
	pkg.path = tmpDir.Join("sharedDir/tmp/pkg")

	job, err := newJob(logger, metrics, chain, pkg, dispatcher, ln, wf)
	assert.NilError(t, err)

	return &testJob{
		job:        job,
		store:      store,
		backend:    backend,
		dispatcher: dispatcher,
	}
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
