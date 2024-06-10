package admin

import (
	"testing"

	"github.com/google/uuid"
	"gotest.tools/v3/assert"
)

func TestPackageName(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	assert.Equal(t, packageName(id, ""), id.String())
	assert.Equal(t, packageName(id, "%sharedPath%watchedDirectories/activeTransfers/standardTransfer/tmp.mCCClmmx0f"), "tmp.mCCClmmx0f")
	assert.Equal(t, packageName(id, "%sharedPath%watchedDirectories/activeTransfers/standardTransfer/tmp.mCCClmmx0f/"), "tmp.mCCClmmx0f")
}
