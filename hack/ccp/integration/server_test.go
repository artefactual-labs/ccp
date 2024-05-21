package integration_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/cenkalti/backoff/v4"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/poll"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	adminv1connect "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1/adminv1beta1connect"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/rootcmd"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/servercmd"
)

func TestServerCreatePackage(t *testing.T) {
	// This test is not going to work until I can have MCPClient connect to CCP.
	//
	// Options:
	// 1. Create an external network in Compose and use host.docker.internal:12345.
	// 2. Set up test using Dagger services.
	// 3. Fake Storage Service
	// t.Skip("Create integration environment.")

	requireFlag(t)
	client := runServer(t)
	restartMCPClient(t)
	ctx := context.Background()

	transferDir := createTransfer(t)

	cpResp, err := client.CreatePackage(ctx, &connect.Request[adminv1.CreatePackageRequest]{
		Msg: &adminv1.CreatePackageRequest{
			Name:        "Foobar",
			Path:        []string{transferDir},
			AutoApprove: &wrapperspb.BoolValue{Value: true},
		},
	})
	assert.NilError(t, err)

	poll.WaitOn(t,
		func(t poll.LogT) poll.Result {
			rpResp, err := client.ReadPackage(ctx, &connect.Request[adminv1.ReadPackageRequest]{
				Msg: &adminv1.ReadPackageRequest{
					Id: cpResp.Msg.Id,
				},
			})
			if err != nil {
				return poll.Error(err)
			}

			pkg := rpResp.Msg.Pkg
			if pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_DONE || pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_COMPLETED_SUCCESSFULLY || pkg.Status == adminv1.PackageStatus_PACKAGE_STATUS_FAILED {
				return poll.Success()
			}

			return poll.Continue("work is still ongoing")
		},
		poll.WithDelay(time.Second/4),
		poll.WithTimeout(time.Second*10),
	)
}

func runServer(t *testing.T) adminv1connect.AdminServiceClient {
	ctx, cancel := context.WithCancel(context.Background())

	dsn := useMySQL(t)

	var sharedDir string
	if useCompose {
		home, _ := os.UserHomeDir()
		sharedDir = filepath.Join(home, ".ccp/am-pipeline-data")
	} else {
		sharedDir = fs.NewDir(t, "amccp-servercmd").Path()
	}

	args := []string{
		// root flags
		"-v=10",
		"--debug",
		// server flags
		"--shared-dir=" + sharedDir,
		"--db.driver=mysql",
		"--db.dsn=" + dsn,
		"--api.admin.addr=:22300",
		"--gearmin.addr=:22301",
		"--ssclient.url=http://127.0.0.1:63081",
		"--ssclient.username=test",
		"--ssclient.key=test",
	}

	var stdout io.Writer
	if useStdout {
		stdout = os.Stdout
	} else {
		stdout = bytes.NewBuffer([]byte{})
	}

	cmd := servercmd.New(&rootcmd.Config{}, stdout)
	assert.NilError(t, cmd.Parse(args))
	done := make(chan error)
	go func() {
		done <- cmd.Exec(ctx, []string{})
	}()

	// Server is likely running, but let's try to receive to see if it failed.
	select {
	case <-time.After(time.Second / 2):
	case err := <-done:
		assert.NilError(t, err)
	}

	t.Cleanup(func() {
		cancel()
		err := <-done
		assert.NilError(t, err)
	})

	baseURL := "http://127.0.0.1:22300"
	waitForHealthStatus(t, baseURL)
	return adminv1connect.NewAdminServiceClient(&http.Client{}, baseURL)
}

// waitForHealthStatus blocks until the heatlh check status succeeds.
func waitForHealthStatus(t *testing.T, baseURL string) {
	client := &http.Client{}

	var retryPolicy backoff.BackOff = backoff.NewConstantBackOff(time.Second)
	retryPolicy = backoff.WithMaxRetries(retryPolicy, 10)

	backoff.RetryNotify(
		func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/grpc.health.v1.Health/Check", bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return errors.New("unexpected status code")
			}

			blob, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if string(blob) != `{"status":"SERVING_STATUS_SERVING"}` {
				return errors.New("unexpected status")
			}

			return nil
		},
		retryPolicy,
		func(err error, d time.Duration) {
			t.Logf("Retrying... (%v)", err)
		},
	)
}

// createTransfer creates a sample transfer in the transfer source directory.
func createTransfer(t *testing.T) string {
	t.Helper()

	const tsRealPath = "/home/archivematica"

	err := os.MkdirAll(filepath.Join(transferSource, "ccp"), os.FileMode(0o770))
	assert.NilError(t, err)

	tmpDir, err := os.MkdirTemp(filepath.Join(transferSource, "ccp"), "transfer-*")
	assert.NilError(t, err)

	writeFile(t, filepath.Join(tmpDir, "f1.txt"), "")
	writeFile(t, filepath.Join(tmpDir, "f2.txt"), "")

	err = os.Link("../hack/processingMCP.xml", filepath.Join(tmpDir, "processingMCP.xml"))
	assert.NilError(t, err)

	tmpDir = strings.TrimPrefix(tmpDir, transferSource)
	tmpDir = filepath.Join(tsRealPath, tmpDir)

	return tmpDir
}

func writeFile(t *testing.T, name, contents string) {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0o644)
	assert.NilError(t, err)

	_, err = file.WriteString(contents)
	assert.NilError(t, err)

	file.Close()
}

func restartMCPClient(t *testing.T) {
	t.Log("Restarting MCPClient...")
	cmd := exec.Command("docker-compose", "restart", "archivematica-mcp-client")
	err := cmd.Run()
	assert.NilError(t, err)
}
