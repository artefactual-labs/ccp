package integration_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

func TestServerCreatePackageViaWatchedDir(t *testing.T) {
	requireFlag(t)
	env := createEnv(t)

	transferName := "Test"
	_ = env.depositTransferViaWatchedDir(
		transferName,
		"standardTransfer",
		workflow.AutomatedConfig,
		configTransformations()...,
	)

	approve(t, env.ctx, env.ccpClient, transferName, adminv1.TransferType_TRANSFER_TYPE_STANDARD)

	poll.WaitOn(t,
		func(t poll.LogT) poll.Result {
			resp, err := env.ccpClient.ListPackages(env.ctx, &connect.Request[adminv1.ListPackagesRequest]{Msg: &adminv1.ListPackagesRequest{
				Type: adminv1.PackageType_PACKAGE_TYPE_TRANSFER,
			}})
			if err != nil {
				return poll.Error(err)
			}

			packages := resp.Msg.Package
			ln := len(packages)
			if ln == 0 {
				return poll.Continue("package not seen yet")
			} else if ln > 2 {
				return poll.Error(fmt.Errorf("unexpected number of packages listed: %d", ln))
			}

			pkg := packages[0]
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_FAILED {
				return poll.Error(errors.New("package processing failed"))
			}
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_DONE || pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_COMPLETED_SUCCESSFULLY {
				return poll.Success()
			}

			return poll.Continue("work is still ongoing")
		},
		poll.WithDelay(time.Second/4),
		poll.WithTimeout(time.Minute*2),
	)
}

func TestServerCreatePackage(t *testing.T) {
	requireFlag(t)
	env := createEnv(t)

	transferDir := env.createTransfer(
		workflow.AutomatedConfig,
		configTransformations()...,
	)

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
		poll.WithTimeout(time.Minute*2),
	)
}

func TestServerCreatePackageWithUserDecision(t *testing.T) {
	requireFlag(t)
	env := createEnv(t)

	transferDir := env.createTransfer(
		workflow.AutomatedConfig,
		configTransformations(
			// Remove "Assign UUIDs to directories" to trigger prompt.
			"bd899573-694e-4d33-8c9b-df0af802437d", "",
		)...,
	)

	cpResp, err := env.ccpClient.CreatePackage(env.ctx, &connect.Request[adminv1.CreatePackageRequest]{
		Msg: &adminv1.CreatePackageRequest{
			Name:        "Foobar",
			Path:        []string{transferDir},
			AutoApprove: &wrapperspb.BoolValue{Value: true},
		},
	})
	assert.NilError(t, err)

	poll.WaitOn(t,
		func(lt poll.LogT) poll.Result {
			rpResp, err := env.ccpClient.ReadPackage(env.ctx, &connect.Request[adminv1.ReadPackageRequest]{
				Msg: &adminv1.ReadPackageRequest{
					Id: cpResp.Msg.Id,
				},
			})
			if err != nil {
				return poll.Error(err)
			}

			pkg := rpResp.Msg.Pkg
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_AWAITING_DECISION {
				for _, decision := range rpResp.Msg.Decision {
					switch decision.Name {
					case "Assign UUIDs to directories?":
						resolve(t, env.ctx, env.ccpClient, decision, "Yes")
						return poll.Continue("decision resolved")
					default:
						return poll.Error(errors.New("unexpected decision to be resolved"))
					}
				}
			}
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_FAILED {
				return poll.Error(errors.New("package processing failed"))
			}
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_DONE || pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_COMPLETED_SUCCESSFULLY {
				return poll.Success()
			}

			return poll.Continue("work is still ongoing")
		},
		poll.WithDelay(time.Second/4),
		poll.WithTimeout(time.Minute),
	)
}
