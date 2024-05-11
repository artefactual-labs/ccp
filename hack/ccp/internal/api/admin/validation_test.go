package admin_test

import (
	"testing"

	"github.com/bufbuild/protovalidate-go"
	"gotest.tools/v3/assert"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

func TestValidation(t *testing.T) {
	t.Parallel()

	v, err := protovalidate.New()
	assert.NilError(t, err)

	req := &adminv1.CreatePackageRequest{
		Name: "asdf",
		Path: []string{"/tmp"},
	}
	err = v.Validate(req)
	assert.NilError(t, err)

	req = &adminv1.CreatePackageRequest{}
	err = v.Validate(req)
	assert.Error(t, err, `validation error:
 - name: value length must be at least 1 characters [string.min_len]
 - path: value must contain at least 1 item(s) [repeated.min_items]`)
}
