package store

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	adminv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

func TestAddDecimalPrecision(t *testing.T) {
	pkg := &adminv1.Package{}
	createdAt := time.Date(2011, 1, 5, 12, 21, 40, 0, time.UTC)
	createdAtDec := "0.123456789"

	// pkg.CreatedAt is a nil pointer, but we're passing the address of the
	// pointer variable instead.
	err := updateTimeWithFraction(&pkg.CreatedAt, createdAt, createdAtDec)

	assert.NilError(t, err)
	assert.Equal(t, pkg.CreatedAt.AsTime().Format(time.RFC3339Nano), "2011-01-05T12:21:40.123456789Z")
}
