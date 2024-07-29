package e2e_test

import (
	"net/http"
	"testing"

	adminv1connect "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1/adminv1beta1connect"
)

func createClient(t *testing.T) adminv1connect.AdminServiceClient {
	t.Helper()

	return adminv1connect.NewAdminServiceClient(&http.Client{}, "http://ccp:8000")
}
