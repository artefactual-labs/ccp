package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	adminv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual-labs/ccp/internal/store/enums"
	sqlc "github.com/artefactual-labs/ccp/internal/store/sqlcmysql"
)

var ErrNotFound error = errors.New("object not found")

type Store interface {
	// RemoveTransientData removes data from the store that the processing
	// engine can't handle after the application is started.
	RemoveTransientData(ctx context.Context) error

	// CreateJob creates a new Job.
	CreateJob(ctx context.Context, params *sqlc.CreateJobParams) error

	// UpdateJobStatus modifies the status of a Job.
	UpdateJobStatus(ctx context.Context, id uuid.UUID, status string) error

	// FindAwaitingJob returns the first job awaiting a decision.
	FindAwaitingJob(ctx context.Context, params *FindAwaitingJobParams) (*adminv1.Job, error)

	// ListJobs returns a list of jobs related to a package showing the most
	// recently created jobs first.
	ListJobs(ctx context.Context, pkgID uuid.UUID) ([]*adminv1.Job, error)

	// CreateTasks creates a group of Tasks in bulk.
	CreateTasks(ctx context.Context, tasks []*Task) error

	// ReadPackagesWithCreationTimestamps returns a list of packages along with
	// their creation timestamps. It excludes hidden packages.
	ReadPackagesWithCreationTimestamps(ctx context.Context, packageType adminv1.PackageType) ([]*adminv1.Package, error)

	// UpdatePackageStatus modifies the status of a Transfer, DIP or SIP.
	UpdatePackageStatus(ctx context.Context, id uuid.UUID, packageType enums.PackageType, status enums.PackageStatus) error

	// ReadTransferLocation returns the current path of a Transfer.
	ReadTransferLocation(ctx context.Context, id uuid.UUID) (loc string, err error)

	// CreateTransfer creates a new transfer not downloaded yet (without path).
	CreateTransfer(ctx context.Context, id uuid.UUID, accessionID, accessSystemID string, metadataSetID uuid.UUID) error

	// ReadTransfer returns a Transfer given its identifier.
	ReadTransfer(ctx context.Context, id uuid.UUID) (transfer Transfer, err error)

	// UpsertTransfer checks for a Transfer using the specified UUID. It updates
	// the current location if the Transfer exists, or it creates a new Transfer
	// with the provided UUID and location if it does not exist.
	UpsertTransfer(ctx context.Context, id uuid.UUID, path string) (created bool, err error)

	// EnsureTransfer checks if a Transfer exists at the given location; creates
	// a new Transfer with a new UUID otherwise.
	EnsureTransfer(ctx context.Context, path string) (id uuid.UUID, created bool, err error)

	// UpdateTransferLocation updates the current location of a given transfer.
	UpdateTransferLocation(ctx context.Context, id uuid.UUID, path string) error

	// ReadSIP returns a SIP given its identifier.
	ReadSIP(ctx context.Context, id uuid.UUID) (sip SIP, err error)

	// UpsertSIP is like UpsertTransfer but targets a SIP instead.
	UpsertSIP(ctx context.Context, id uuid.UUID, path string) (created bool, err error)

	// EnsureTransfer is like EnsureTransfer but targets a SIP instead.
	EnsureSIP(ctx context.Context, path string) (id uuid.UUID, created bool, err error)

	// ReadDIP returns a DIP given its identifier.
	ReadDIP(ctx context.Context, id uuid.UUID) (dip DIP, err error)

	// UpsertDIP is like UpsertTransfer but targets a SIP instead.
	UpsertDIP(ctx context.Context, id uuid.UUID, path string) (created bool, err error)

	// EnsureDIP is like EnsureTransfer but targets a SIP instead.
	EnsureDIP(ctx context.Context, path string) (id uuid.UUID, created bool, err error)

	// ReadUnitVars retrieves a list of package variables associated with a
	// specific package identified by its type and UUID. It filters the
	// variables based on the provided name. If name is an empty string, it
	// returns all variables for the specified package. If name is provided,
	// only variables matching this name are returned.
	ReadUnitVars(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name string) ([]UnitVar, error)

	// ReadUnitVar reads a string value stored as a package variable.
	ReadUnitVar(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name string) (string, error)

	// ReadUnitLinkID reads a workflow link ID stored as a package variable.
	ReadUnitLinkID(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name string) (uuid.UUID, error)

	// CreateUnitVar creates a new variable.
	CreateUnitVar(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name, value string, linkID uuid.UUID, update bool) error

	// Files returns a list of files. This could return some kind of iterator
	// interface; rangefunc did work but it's not supported by linters yet.
	Files(ctx context.Context, id uuid.UUID, packageType enums.PackageType, filterFilenameEnd, filterSubdir, replacementPath string) ([]File, error)

	// ReadPipelineID reads the identifier of this pipeline.
	ReadPipelineID(ctx context.Context) (uuid.UUID, error)

	// ReadDict reads a dictionary given its name.
	ReadDict(ctx context.Context, name string) (map[string]string, error)

	// ValidateUserAPIKey checks if a user with the given username and API key
	// exists and is active. It returns a pointer to the User if valid, or nil
	// and an error otherwise. A nil User doesn't necessarily mean the user
	// doesn't exist; check the error for details.
	ValidateUserAPIKey(ctx context.Context, username, key string) (*User, error)

	Running() bool
	Close() error
}

func New(logger logr.Logger, driver, dsn string) (Store, error) {
	var store *mysqlStoreImpl

	switch strings.ToLower(driver) {
	case "mysql":
		{
			logger = logger.WithName("mysql")
			pool, err := connectToMySQL(logger, dsn)
			if err != nil {
				return nil, fmt.Errorf("connect to MySQL: %v", err)
			}
			store, err = newMySQLStore(logger, pool)
			if err != nil {
				return nil, fmt.Errorf("new MySQL store: %v", err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported db driver: %q", driver)
	}

	return store, nil
}

type UnitVar struct {
	Name   string
	Value  *string
	LinkID *uuid.UUID
}

type Transfer struct {
	ID          uuid.UUID
	Name        string
	CurrentPath string
	Type        adminv1.TransferType
	Status      adminv1.PackageStatus
}

type SIP struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	CurrentPath string
	Hidden      bool
	AIPFilename string
	Type        string // SIP, AIC, AIP-REIN, AIC-REIN
	DirIDs      bool
	Status      int
	CompletedAt time.Time
}

type DIP struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	CurrentPath string
	Hidden      bool
	AIPFilename string
	DirIDs      bool
	Status      int
	CompletedAt time.Time
}

type File struct {
	ID               uuid.UUID `db:"fileUUID"`
	CurrentLocation  string    `db:"currentLocation"`
	OriginalLocation string    `db:"originalLocation"`
	FileGrpUse       string    `db:"fileGrpUse"`
}

type Task struct {
	ID        uuid.UUID     `db:"taskUUID"`
	CreatedAt time.Time     `db:"createdTime"`
	FileID    uuid.NullUUID `db:"fileUUID"`
	Filename  string        `db:"fileName"`
	Exec      string        `db:"exec"`
	Arguments string        `db:"arguments"`
	StartedAt sql.NullTime  `db:"startTime"`
	EndedAt   sql.NullTime  `db:"endTime"`
	Client    string        `db:"client"`
	Stdout    string        `db:"stdOut"`
	Stderr    string        `db:"stdError"`
	ExitCode  sql.NullInt16 `db:"exitCode"`
	JobID     uuid.UUID     `db:"jobuuid"`
}

type FindAwaitingJobParams struct {
	Directory *string
	PackageID *uuid.UUID
	Group     *string
}

type User struct {
	ID       int
	Username string
	Email    string
	Active   bool
	AgentID  *int
}
