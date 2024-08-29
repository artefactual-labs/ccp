package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"dagger/ccp/internal/dagger"
)

const (
	dbName     = "CCP"
	dbDumpsDir = "e2e/testdata/dumps"
)

// Options for the shared Archivematica directory that we provisiong using
// cache volumes.
var (
	sharedDir                = "/var/archivematica/sharedDirectory"
	sharedDirVolume          = dag.CacheVolume("share")
	sharedDirVolumeMountOpts = dagger.ContainerWithMountedCacheOpts{
		Sharing: dagger.Shared,
		Owner:   "1000:1000",
	}
)

// DatabaseExecutionMode defines the different modes in which the e2e tests can
// operate with the application databases.
type DatabaseExecutionMode string

const (
	// UseDumps attempts to configure the MySQL service using the database dumps
	// previously generated.
	UseDumps DatabaseExecutionMode = "USE_DUMPS"
	// UseCached is the default mode that relies on whatever is the existing
	// MySQL service state.
	UseCached DatabaseExecutionMode = "USE_CACHED"
	// ForceDrop drops the existing databases forcing the application to
	// recreate them using Django migrations.
	ForceDrop DatabaseExecutionMode = "FORCE_DROP"
)

func (m *CCP) GenerateDumps(ctx context.Context) (*dagger.Directory, error) {
	mysql := m.Build().MySQLContainer().AsService()

	// ForceDrop ensures that the app is migrated and installed.
	if err := m.populateDatabase(ctx, mysql, ForceDrop); err != nil {
		return nil, err
	}

	return dumpDB(mysql, dbName)
}

// Run the e2e tests.
//
// This function configures
func (m *CCP) Etoe(
	ctx context.Context,
	// +optional
	test string,
	// +default="USE_DUMPS"
	dbMode DatabaseExecutionMode,
) error {
	mysql := m.Build().MySQLContainer().
		WithMountedCache("/var/lib/mysql", dag.CacheVolume("mysql")).
		AsService()

	if err := m.populateDatabase(ctx, mysql, dbMode); err != nil {
		return err
	}

	ccp := m.bootstrapCCP(mysql)

	worker := m.bootstrapWorker(mysql, ccp)

	args := []string{"go", "test", "-v"}
	{
		if test != "" {
			args = append(args, "-run", fmt.Sprintf("Test%s", test))
		}
		args = append(args, "./e2e/...")
	}

	dag.Go(dagger.GoOpts{
		Container: goModule().
			WithSource(m.Source).
			Container().
			WithServiceBinding("mysql", mysql).
			WithServiceBinding("ccp", ccp).
			WithServiceBinding("worker", worker).
			WithMountedCache(sharedDir, sharedDirVolume, sharedDirVolumeMountOpts).
			WithEnvVariable("CCP_E2E_ENABLED", "yes"),
	}).
		Exec(args).
		Stdout(ctx)

	return nil
}

func (m *CCP) bootstrapCCP(mysql *dagger.Service) *dagger.Service {
	return m.Build().CCPImage().
		WithEnvVariable("CCP_DEBUG", "true").
		WithEnvVariable("CCP_V", "10").
		WithEnvVariable("CCP_SHARED_DIR", sharedDir).
		WithEnvVariable("CCP_DB_DRIVER", "mysql").
		WithEnvVariable("CCP_DB_DSN", "root:12345@tcp(mysql:3306)/CCP").
		WithEnvVariable("CCP_API_ADMIN_ADDR", ":8000").
		WithEnvVariable("CCP_WEBUI_ADDR", ":8001").
		WithEnvVariable("CCP_METRICS_ADDR", ":7999").
		WithServiceBinding("mysql", mysql).
		WithMountedCache(sharedDir, sharedDirVolume, sharedDirVolumeMountOpts).
		AsService()
}

func (m *CCP) populateDatabase(ctx context.Context, mysql *dagger.Service, dbMode DatabaseExecutionMode) error {
	drop := dbMode != UseCached
	if err := createDB(ctx, mysql, dbName, drop); err != nil {
		return err
	}

	if dbMode == UseDumps {
		dumpFile := m.Source.File(filepath.Join(dbDumpsDir, fmt.Sprintf("%s.sql.bz2", dbName)))
		if err := loadDump(ctx, mysql, dbName, dumpFile); err != nil {
			return err
		}
	}

	ctr := m.Build().WorkerImage().
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_USER", "root").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_PASSWORD", "12345").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_HOST", "mysql").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_DATABASE", dbName).
		WithServiceBinding("mysql", mysql).
		WithEnvVariable("CACHEBUSTER", time.Now().Format(time.RFC3339Nano))

	if _, err := ctr.
		WithExec([]string{"/src/manage.py", "migrate", "--noinput"}).
		Sync(ctx); err != nil {
		return err
	}

	onlyMigrate := dbMode == UseDumps || dbMode == UseCached
	if onlyMigrate {
		return nil
	}

	if _, err := ctr.
		WithExec([]string{
			"/src/manage.py", "install",
			`--username=test`,
			`--password=test`,
			`--email=test@test.com`,
			`--org-name=test`,
			`--org-id=test`,
			`--api-key=test`,
			`--site-url=http://todo:8000`,
		}).
		Sync(ctx); err != nil {
		return err
	}

	return nil
}

func (m *CCP) bootstrapWorker(mysql, ccp *dagger.Service) *dagger.Service {
	return m.Build().WorkerImage().
		WithEnvVariable("DJANGO_SECRET_KEY", "12345").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_USER", "root").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_PASSWORD", "12345").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_HOST", "mysql").
		WithEnvVariable("ARCHIVEMATICA_WORKER_DB_DATABASE", dbName).
		WithEnvVariable("ARCHIVEMATICA_WORKER_GEARMAN_SERVER", "ccp:4730").
		WithMountedCache(sharedDir, sharedDirVolume, sharedDirVolumeMountOpts).
		WithServiceBinding("mysql", mysql).
		WithServiceBinding("ccp", ccp).
		AsService()
}

func createDB(ctx context.Context, mysql *dagger.Service, dbname string, drop bool) error {
	if drop {
		if err := mysqlCommand(ctx, mysql, "DROP DATABASE IF EXISTS "+dbname); err != nil {
			return err
		}
	}

	return mysqlCommand(ctx, mysql, "CREATE DATABASE IF NOT EXISTS "+dbname)
}

func mysqlCommand(ctx context.Context, mysql *dagger.Service, cmd string) error {
	_, err := dag.Container().
		From(mysqlImage).
		WithEnvVariable("CACHEBUSTER", time.Now().Format(time.RFC3339Nano)).
		WithServiceBinding("mysql", mysql).
		WithExec([]string{"mysql", "-hmysql", "-uroot", "-p12345", "-e", cmd}).
		Sync(ctx)

	return err
}

func loadDump(ctx context.Context, mysql *dagger.Service, dbname string, dump *dagger.File) error {
	_, err := dag.Container().
		From(mysqlImage).
		WithEnvVariable("CACHEBUSTER", time.Now().Format(time.RFC3339Nano)).
		WithServiceBinding("mysql", mysql).
		WithFile("/tmp/dump.sql.bz2", dump).
		WithExec([]string{"/bin/sh", "-c", "bunzip2 < /tmp/dump.sql.bz2 | mysql -hmysql -uroot -p12345 " + dbname}).
		Sync(ctx)

	return err
}

func dumpDB(mysql *dagger.Service, dbs ...string) (*dagger.Directory, error) {
	ctr := dag.Container().
		From(mysqlImage).
		WithEnvVariable("CACHEBUSTER", time.Now().Format(time.RFC3339Nano)).
		WithServiceBinding("mysql", mysql).
		WithExec([]string{"mkdir", "/tmp/dumps"}).
		WithWorkdir("/tmp/dumps")

	for _, dbname := range dbs {
		ctr = ctr.WithExec([]string{
			"/bin/sh", "-c",
			fmt.Sprintf("mysqldump -hmysql -uroot -p12345 %s | bzip2 -c > %s", dbname, fmt.Sprintf("%s.sql.bz2", dbname)),
		})
	}

	return ctr.Directory("/tmp/dumps"), nil
}
