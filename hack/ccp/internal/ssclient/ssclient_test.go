package ssclient_test

import (
	"testing"

	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/fake"
)

func TestClient(t *testing.T) {
	t.Parallel()

	store := fake.NewMockStore(gomock.NewController(t))

	c, err := ssclient.NewClient(store, "bu", "u", "k")
	assert.NilError(t, err)
	assert.Assert(t, c != nil)
}
