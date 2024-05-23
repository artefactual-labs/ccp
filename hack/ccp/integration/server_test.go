package integration_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

func TestServerCreatePackage(t *testing.T) {
	requireFlag(t)
	env := createEnv(t)

	transferDir := env.createTransfer()

	cpResp, err := env.ccpClient.CreatePackage(env.ctx, &connect.Request[adminv1.CreatePackageRequest]{
		Msg: &adminv1.CreatePackageRequest{
			Name:        "Foobar",
			Path:        []string{transferDir},
			AutoApprove: &wrapperspb.BoolValue{Value: true},
		},
	})
	assert.NilError(t, err)

	poll.WaitOn(t,
		func(t poll.LogT) poll.Result {
			rpResp, err := env.ccpClient.ReadPackage(env.ctx, &connect.Request[adminv1.ReadPackageRequest]{
				Msg: &adminv1.ReadPackageRequest{
					Id: cpResp.Msg.Id,
				},
			})
			if err != nil {
				return poll.Error(err)
			}

			pkg := rpResp.Msg.Pkg
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_FAILED {
				return poll.Error(err)
			}
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_DONE || pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_COMPLETED_SUCCESSFULLY {
				return poll.Success()
			}

			return poll.Continue("work is still ongoing")
		},
		poll.WithDelay(time.Second/4),
		poll.WithTimeout(time.Second*120),
	)

	t.Log("Test completed successfully!")
}
