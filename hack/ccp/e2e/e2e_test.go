package e2e_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

func TestPrometheusServer(t *testing.T) {
	requireFlag(t)

	t.Run("Exposes metrics via its HTTP API", func(t *testing.T) {
		resp, err := http.Get("http://ccp:7999/metrics")
		assert.NilError(t, err)
		defer resp.Body.Close()

		blob, err := io.ReadAll(resp.Body)
		assert.NilError(t, err)

		assert.Assert(t, cmp.Contains(string(blob), "mcpserver_active_jobs 0"))
	})
}

func TestServerCreatePackage(t *testing.T) {
	requireFlag(t)

	t.Run("Test", func(t *testing.T) {
		client := createClient(t)

		req := &connect.Request[adminv1.ListPackagesRequest]{
			Msg: &adminv1.ListPackagesRequest{
				Type: adminv1.PackageType_PACKAGE_TYPE_SIP,
			},
		}
		req.Header().Set("Authorization", "ApiKey test:test")

		resp, err := client.ListPackages(context.Background(), req)
		assert.NilError(t, err)

		t.Log(resp.Msg.Package)
	})
}
