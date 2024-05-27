package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"go.uber.org/goleak"
	"golang.org/x/exp/slices"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/integration/storage"
	adminv1connect "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1/adminv1beta1connect"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/rootcmd"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/servercmd"
)

const envPrefix = "CCP_INTEGRATION"

var (
	enableIntegration           = getEnvBool("ENABLED", "no")
	enableLogging               = getEnvBool("ENABLE_LOGGING", "yes")
	enableTestContainersLogging = getEnvBool("ENABLE_TESTCONTAINERS_LOGGING", "no")
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type hostPort struct {
	host  string
	port  string
	portN int
}

func newHostPort(host string, port int) hostPort {
	return hostPort{host, strconv.Itoa(port), port}
}

func (hp hostPort) String() string {
	return hp.Addr()
}

func (hp hostPort) Addr() string {
	return net.JoinHostPort(hp.host, hp.port)
}

func (hp hostPort) URL(scheme string) string {
	u, _ := url.Parse(scheme + "://" + hp.Addr())
	return u.String()
}

type env struct {
	t   testing.TB
	ctx context.Context

	// Information about the current user that is running this test.
	user *user.User

	// Path to the shared directory in the local filesystem.
	sharedDir string

	// Path to the transfer source directory in the local filesystem.
	transferSourceDir string

	// Listen address for the CCP Admin API server.
	// TODO: use free port.
	ccpAdminServerAddr hostPort

	// Listen address for the CCP Job server.
	// TODO: use free port.
	ccpJobServerAddr hostPort

	// CCP Admin API client, should be ready to use once the env is created.
	ccpClient adminv1connect.AdminServiceClient

	storageServiceBaseURL string

	// MySQL client and connection details.
	mysqlClient      *sql.DB
	mysqlDSN         string
	mysqlContainerIP string
}

// createEnv brings up all the dependencies needed by CCP to run our integration
// tests successfully. It uses testcontainers to run containers.
func createEnv(t *testing.T) *env {
	env := &env{
		t:                  t,
		ctx:                context.Background(),
		ccpAdminServerAddr: newHostPort("127.0.0.1", 22300),
		ccpJobServerAddr:   newHostPort("127.0.0.1", 22301),
	}
	env.lookUpUser()

	testcontainers.Logger = logger{t}

	env.sharedDir = env.tempDir("ccp-sharedDir")
	env.transferSourceDir = env.tempDir("ccp-transferSourceDir")

	// These are all blocking.
	env.runMySQL()
	env.runStorageService()
	env.runCCP()
	env.runMCPClient()

	return env
}

func (e *env) tempDir(name string) string {
	tmpDir, err := os.MkdirTemp("", name+"-*")
	assert.NilError(e.t, err)
	return tmpDir
}

func (e *env) lookUpUser() {
	e.t.Log("Looking up user...")

	user, err := user.Current()
	assert.NilError(e.t, err)
	e.user = user
}

func (e *env) runMySQL() {
	e.t.Log("Running MySQL server...")

	container, err := mysql.RunContainer(e.ctx,
		testcontainers.WithImage("mysql:8.4.0"),
		testcontainers.CustomizeRequestOption(func(req *testcontainers.GenericContainerRequest) error {
			req.LogConsumerCfg = &testcontainers.LogConsumerConfig{
				Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
				Consumers: []testcontainers.LogConsumer{&logConsumer{e.t, "mysql"}},
			}
			return nil
		}),
		mysql.WithDatabase("MCP"),
		mysql.WithUsername("root"),
		mysql.WithPassword("12345"),
		mysql.WithScripts("data/mcp.sql.bz2"),
	)
	assert.NilError(e.t, err, "Failed to start container.")
	e.t.Cleanup(func() {
		_ = container.Terminate(e.ctx)
	})

	e.mysqlContainerIP, err = container.ContainerIP(e.ctx)
	assert.NilError(e.t, err)

	e.mysqlDSN, err = container.ConnectionString(e.ctx)
	assert.NilError(e.t, err, "Failed to create connection string to MySQL server.")

	e.mysqlClient, err = sql.Open("mysql", e.mysqlDSN)
	assert.NilError(e.t, err, "Failed to connect to MySQL.")
	e.t.Cleanup(func() { e.mysqlClient.Close() })

	err = e.mysqlClient.Ping()
	assert.NilError(e.t, err, "Failed to ping MySQL.")
}

func (e *env) runStorageService() {
	e.t.Log("Running Storage Service stub...")

	srv := storage.New(e.t, e.sharedDir, e.transferSourceDir)
	e.storageServiceBaseURL = srv.URL
}

func (e *env) runMCPClient() {
	e.t.Log("Running Archivematica MCPClient...")

	// Update the database with the URL of the Storage Service stub.
	u, err := url.Parse(e.storageServiceBaseURL)
	assert.NilError(e.t, err)
	ssPort, err := strconv.Atoi(u.Port())
	assert.NilError(e.t, err)
	u.Host = fmt.Sprintf("%s:%d", testcontainers.HostInternal, ssPort)
	_, err = e.mysqlClient.ExecContext(e.ctx, "UPDATE DashboardSettings SET value = ? WHERE name = 'storage_service_url';", u.String())
	assert.NilError(e.t, err)

	req := testcontainers.ContainerRequest{
		Name: "ccp-archivematica-mcp-client",
		FromDockerfile: testcontainers.FromDockerfile{
			// We could start using a public image instead once 1917 and 1931 are merged.
			Context:       "../../../",
			Dockerfile:    "hack/ccp/integration/data/Dockerfile.worker",
			PrintBuildLog: false,
			KeepImage:     true,
		},
		HostAccessPorts: []int{
			e.ccpJobServerAddr.portN, // Proxy Gearmin job server.
			ssPort,                   // Proxy Storage server.
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Mounts = []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   e.sharedDir,
					Target:   "/var/archivematica/sharedDirectory",
					ReadOnly: false,
				},
			}
		},
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{&logConsumer{e.t, "worker"}},
		},
		Env: map[string]string{
			"DJANGO_SECRET_KEY":                                        "12345",
			"ARCHIVEMATICA_MCPCLIENT_CLIENT_USER":                      "root",
			"ARCHIVEMATICA_MCPCLIENT_CLIENT_PASSWORD":                  "12345",
			"ARCHIVEMATICA_MCPCLIENT_CLIENT_HOST":                      e.mysqlContainerIP,
			"ARCHIVEMATICA_MCPCLIENT_CLIENT_PORT":                      "3306",
			"ARCHIVEMATICA_MCPCLIENT_CLIENT_DATABASE":                  "MCP",
			"ARCHIVEMATICA_MCPCLIENT_MCPCLIENT_MCPARCHIVEMATICASERVER": fmt.Sprintf("%s:%d", testcontainers.HostInternal, e.ccpJobServerAddr.portN),
			"ARCHIVEMATICA_MCPCLIENT_MCPCLIENT_SEARCH_ENABLED":         "false",
		},
	}

	container, err := testcontainers.GenericContainer(e.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NilError(e.t, err)

	e.t.Cleanup(func() {
		if e.t.Failed() {
			e.logContainerOutput(container)
		}

		_ = container.Terminate(e.ctx)
	})
}

func (e *env) logContainerOutput(container testcontainers.Container) {
	reader, err := container.Logs(e.ctx)
	assert.NilError(e.t, err)

	blob, err := io.ReadAll(reader)
	assert.NilError(e.t, err)

	e.t.Log(string(blob))
}

func (e *env) runCCP() {
	ctx, cancel := context.WithCancel(e.ctx)

	args := []string{
		// root flags
		"-v=10",
		"--debug",
		// server flags
		"--shared-dir=" + e.sharedDir,
		"--db.driver=mysql",
		"--db.dsn=" + e.mysqlDSN,
		"--api.admin.addr=" + e.ccpAdminServerAddr.Addr(),
		"--gearmin.addr=" + e.ccpJobServerAddr.Addr(),
		"--ssclient.url=" + e.storageServiceBaseURL,
		"--ssclient.username=test",
		"--ssclient.key=test",
	}

	var stdout io.Writer
	if enableLogging {
		stdout = os.Stdout
	} else {
		stdout = bytes.NewBuffer([]byte{})
	}

	cmd := servercmd.New(&rootcmd.Config{}, stdout)
	assert.NilError(e.t, cmd.Parse(args))
	done := make(chan error)
	go func() {
		done <- cmd.Exec(ctx, []string{})
	}()

	// Server is likely running, but let's try to receive to see if it failed.
	select {
	case <-time.After(time.Second / 2):
	case err := <-done:
		assert.NilError(e.t, err)
	}

	e.t.Cleanup(func() {
		cancel()
		err := <-done
		assert.NilError(e.t, err)
	})

	baseURL := e.ccpAdminServerAddr.URL("http")
	waitForHealthStatus(e.t, baseURL)

	e.ccpClient = adminv1connect.NewAdminServiceClient(&http.Client{}, baseURL)
}

// createTransfer creates a sample transfer in the transfer source directory.
func (e *env) createTransfer() string {
	tmpDir, err := os.MkdirTemp(e.transferSourceDir, "transfer-*")
	assert.NilError(e.t, err)

	writeFile(e.t, filepath.Join(tmpDir, "f1.txt"), "")
	writeFile(e.t, filepath.Join(tmpDir, "f2.txt"), "")

	err = os.Link("../hack/processingMCP.xml", filepath.Join(tmpDir, "processingMCP.xml"))
	assert.NilError(e.t, err)

	e.t.Logf("Created transfer: %s", tmpDir)

	return tmpDir
}

// waitForHealthStatus blocks until the heatlh check status succeeds.
func waitForHealthStatus(t testing.TB, baseURL string) {
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

func writeFile(t testing.TB, name, contents string) {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0o644)
	assert.NilError(t, err)

	_, err = file.WriteString(contents)
	assert.NilError(t, err)

	file.Close()
}

func getEnv(name, fallback string) string {
	v := os.Getenv(fmt.Sprintf("%s_%s", envPrefix, name))
	if v == "" {
		return fallback
	}
	return v
}

func getEnvRequired(name string) string { //nolint: unused
	v := getEnv(name, "")
	if v == "" && enableIntegration {
		log.Fatalf("Required env %s_%s is empty.", envPrefix, name)
	}
	return v
}

func getEnvBool(name, fallback string) bool {
	if v := getEnv(name, fallback); slices.Contains([]string{"yes", "1", "on", "true"}, v) {
		return true
	} else {
		return false
	}
}

func requireFlag(t *testing.T) {
	if !enableIntegration {
		t.Skip("Skipping integration tests (CCP_INTEGRATION_ENABLED=no).")
	}
}

// logger implements testcontainers.Logging. This implementation logs only if
// requested by the user via enableTestContainersLogging.
type logger struct {
	testing.TB
}

func (l logger) Printf(format string, v ...interface{}) {
	if enableTestContainersLogging {
		l.Logf(format, v...)
	}
}

// logConsumer implements testcontainers.LogConsumer.
type logConsumer struct {
	t         testing.TB
	container string
}

func (c *logConsumer) Accept(l testcontainers.Log) {
	if enableTestContainersLogging {
		content := string(l.Content)
		content = strings.TrimSuffix(content, "\n")
		c.t.Logf("[%s] %s", c.container, content)
	}
}
