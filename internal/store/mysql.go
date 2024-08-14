package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/go-logr/logr"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"go.artefactual.dev/tools/ref"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual-labs/ccp/internal/store/enums"
	sqlc "github.com/artefactual-labs/ccp/internal/store/sqlcmysql"
)

var (
	myJobsTable  = "Jobs"
	myFilesTable = "Files"
)

func connectToMySQL(logger logr.Logger, dsn string) (*sql.DB, error) {
	config, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing dsn: %v (%s)", err, dsn)
	}
	config.Collation = "utf8mb4_unicode_ci"
	config.Loc = time.UTC
	config.ParseTime = true
	config.MultiStatements = true
	config.Params = map[string]string{
		"time_zone": "'+00:00'",
	}

	conn, err := mysqldriver.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("error creating connector: %w", err)
	}

	db := sql.OpenDB(conn)

	// Set reasonable sizes on the built-in pool.
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(30)
	db.SetConnMaxLifetime(time.Minute)

	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return nil, err
	}

	logger.V(2).Info("Connected to MySQL.", "version", version)

	return db, nil
}

// mysqlstoreImpl implements the Store interface. While most queries are built
// using sqlc, there are some cases where more dynamism is required where we
// are using the goqu SQL builder, e.g. UpdatePackageStatus.
type mysqlStoreImpl struct {
	logger  logr.Logger
	pool    *sql.DB
	queries *sqlc.Queries
	goqu    *goqu.Database
}

var _ Store = (*mysqlStoreImpl)(nil)

func newMySQLStore(logger logr.Logger, pool *sql.DB) (*mysqlStoreImpl, error) {
	queries, err := sqlc.Prepare(context.Background(), pool)
	if err != nil {
		return nil, err
	}

	return &mysqlStoreImpl{
		logger:  logger,
		pool:    pool,
		queries: queries,
		goqu:    goqu.New("mysql", pool),
	}, nil
}

func (s *mysqlStoreImpl) RemoveTransientData(ctx context.Context) (err error) {
	defer wrap(&err, "RemoveTransientData")

	conn, _ := s.pool.Conn(ctx)
	defer conn.Close()

	q := sqlc.New(conn)

	// TODO: lock database?

	if err := q.CleanUpActiveTasks(ctx); err != nil {
		return err
	}

	if err := q.CleanUpActiveTransfers(ctx); err != nil {
		return err
	}

	if err := q.CleanUpTasksWithAwaitingJobs(ctx); err != nil {
		return err
	}

	if err := q.CleanUpAwaitingJobs(ctx); err != nil {
		return err
	}

	if err := q.CleanUpActiveSIPs(ctx); err != nil {
		return err
	}

	if err := q.CleanUpActiveJobs(ctx); err != nil {
		return err
	}

	return nil
}

func (s *mysqlStoreImpl) CreateJob(ctx context.Context, params *sqlc.CreateJobParams) (err error) {
	defer wrap(&err, "CreateJob")

	return s.queries.CreateJob(ctx, params)
}

func (s *mysqlStoreImpl) UpdateJobStatus(ctx context.Context, id uuid.UUID, status string) (err error) {
	defer wrap(&err, "UpdateJobStatus(%s, %s)", id, status)

	var step int32
	switch status {
	case "Unknown", "STATUS_UNKNOWN", "":
		step = 0
	case "Awaiting decision", "STATUS_AWAITING_DECISION":
		step = 1
	case "Completed successfully", "STATUS_COMPLETED_SUCCESSFULLY":
		step = 2
	case "Executing command(s)", "STATUS_EXECUTING_COMMANDS":
		step = 3
	case "Failed", "STATUS_FAILED":
		step = 4
	default:
		return fmt.Errorf("unknown status: %q", status)
	}

	return s.queries.UpdateJobStatus(ctx, &sqlc.UpdateJobStatusParams{
		ID:          id,
		Currentstep: step,
	})
}

func (s *mysqlStoreImpl) FindAwaitingJob(ctx context.Context, params *FindAwaitingJobParams) (_ *adminv1.Job, err error) {
	defer wrap(&err, "FindAwaitingJob(ctx, params)")

	ex := goqu.Ex{"currentStep": adminv1.JobStatus_JOB_STATUS_AWAITING_DECISION}

	if params.Directory != nil { // ApproveTransferByPath
		ex["directory"] = *params.Directory
	} else if params.PackageID != nil { // ApprovePartialReingest
		ex["SIPUUID"] = params.PackageID.String()
		ex["microserviceGroup"] = ref.DerefZero(params.Group)
	}

	sel := s.goqu.Select().From(myJobsTable).Where(ex).Limit(1)

	j := struct {
		ID        uuid.UUID `db:"jobUUID"`
		PackageID uuid.UUID `db:"SIPUUID"`
	}{}
	if ok, err := sel.ScanStructContext(ctx, &j); err != nil {
		return nil, fmt.Errorf("scan: %v", err)
	} else if !ok {
		return nil, ErrNotFound
	}

	ret := &adminv1.Job{
		Id:        j.ID.String(),
		PackageId: j.PackageID.String(),
	}

	return ret, nil
}

func (s *mysqlStoreImpl) ListJobs(ctx context.Context, pkgID uuid.UUID) (_ []*adminv1.Job, err error) {
	defer wrap(&err, "ListJobs(tasks)")

	jobs, err := s.queries.ListJobs(ctx, pkgID)
	if err != nil {
		return nil, err
	}

	convert := func(j *sqlc.Job) (*adminv1.Job, error) {
		ret := &adminv1.Job{
			Id:              j.ID.String(),
			PackageId:       j.SIPID.String(),
			Directory:       j.Directory,
			LinkId:          j.LinkID.UUID.String(),
			LinkDescription: j.Type,
			Hidden:          j.Hidden,
			Group:           j.Microservicegroup,
			Status:          adminv1.JobStatus(j.Currentstep),
		}

		switch j.Unittype {
		case "unitDIP":
			ret.PackageType = adminv1.PackageType_PACKAGE_TYPE_DIP
		case "unitSIP":
			ret.PackageType = adminv1.PackageType_PACKAGE_TYPE_SIP
		case "unitTransfer":
			ret.PackageType = adminv1.PackageType_PACKAGE_TYPE_TRANSFER
		}

		if err := updateTimeWithFraction(&ret.CreatedAt, j.CreatedAt, j.Createdtimedec); err != nil {
			return nil, err
		}

		return ret, nil
	}

	ret := make([]*adminv1.Job, 0, len(jobs))
	for _, item := range jobs {
		if j, err := convert(item); err != nil {
			return nil, fmt.Errorf("convert: %v", err)
		} else {
			ret = append(ret, j)
		}
	}

	return ret, nil
}

func (s *mysqlStoreImpl) CreateTasks(ctx context.Context, tasks []*Task) (err error) {
	defer wrap(&err, "CreateTasks(tasks)")

	insert := s.goqu.Insert("Tasks").Rows(tasks).Executor()
	if _, err := insert.ExecContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *mysqlStoreImpl) ReadPackagesWithCreationTimestamps(ctx context.Context, packageType adminv1.PackageType) (ret []*adminv1.Package, err error) {
	defer wrap(&err, "ReadPackagesWithCreationTimestamps(tasks)")

	switch packageType {
	case adminv1.PackageType_PACKAGE_TYPE_TRANSFER:
		rows, err := s.queries.ListTransfersWithCreationTimestamps(ctx)
		if err != nil {
			return nil, err
		}
		ret = make([]*adminv1.Package, 0, len(rows))
		for _, row := range rows {
			pkg := &adminv1.Package{}
			pkg.Id = row.SIPID.String()
			pkg.Status = adminv1.PackageStatus(int32(row.Status.Int16))
			if err := updateTimeWithFraction(&pkg.CreatedAt, row.CreatedAt, row.CreatedAtDec); err != nil {
				return nil, err
			}
			ret = append(ret, pkg)
		}
	case adminv1.PackageType_PACKAGE_TYPE_SIP:
		rows, err := s.queries.ListSIPsWithCreationTimestamps(ctx)
		if err != nil {
			return nil, err
		}
		ret = make([]*adminv1.Package, 0, len(rows))
		for _, row := range rows {
			pkg := &adminv1.Package{}
			pkg.Id = row.SIPID.String()
			pkg.Status = adminv1.PackageStatus(int32(row.Status.Int16))
			if err := updateTimeWithFraction(&pkg.CreatedAt, row.CreatedAt, row.CreatedAtDec); err != nil {
				return nil, err
			}
			ret = append(ret, pkg)
		}
	default:
		return nil, fmt.Errorf("unsupported package type: %s", packageType)
	}

	return ret, nil
}

func (s *mysqlStoreImpl) UpdatePackageStatus(ctx context.Context, id uuid.UUID, packageType enums.PackageType, status enums.PackageStatus) (err error) {
	defer wrap(&err, "UpdatePackageStatus(%s, %s, %s)", id, packageType, status)

	if !packageType.IsValid() {
		return fmt.Errorf("invalid type: %v", err)
	}
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %v", err)
	}

	var (
		table    string
		idColumn string
	)
	switch packageType {
	case enums.PackageTypeTransfer:
		table = "Transfers"
		idColumn = "transferUUID"
	case enums.PackageTypeDIP, enums.PackageTypeSIP:
		table = "SIPs"
		idColumn = "sipUUID"
	default:
		return fmt.Errorf("unknown unit type: %q", packageType)
	}

	values := goqu.Record{
		"status": int(status),
	}
	if status == enums.PackageStatusCompletedSuccessfully {
		values["completed_at"] = time.Now()
	}

	update := s.goqu.Update(table).
		Where(goqu.Ex{idColumn: id.String()}).
		Set(values).
		Executor()

	_, err = update.ExecContext(ctx)

	return err
}

func (s *mysqlStoreImpl) ReadTransferLocation(ctx context.Context, id uuid.UUID) (loc string, err error) {
	defer wrap(&err, "ReadTransferLocation(%s)", id)

	ret, err := s.queries.ReadTransferLocation(ctx, id)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return ret.Currentlocation, nil
}

func (s *mysqlStoreImpl) CreateTransfer(ctx context.Context, id uuid.UUID, accessionID, accessSystemID string, metadataSetID uuid.UUID) (err error) {
	defer wrap(&err, "CreateTransfer(%s, %s, %s, %d)", id, accessionID, accessSystemID, metadataSetID)

	params := &sqlc.CreateTransferParams{
		Transferuuid:   id,
		Accessionid:    accessionID,
		AccessSystemID: accessSystemID,
	}
	if metadataSetID != uuid.Nil {
		params.Transfermetadatasetrowuuid = uuid.NullUUID{
			UUID:  metadataSetID,
			Valid: true,
		}
	}

	return s.queries.CreateTransfer(ctx, params)
}

func (s *mysqlStoreImpl) ReadTransfer(ctx context.Context, id uuid.UUID) (_ Transfer, err error) {
	defer wrap(&err, "ReadTransfer(%s)", id)

	transfer := Transfer{}

	row, err := s.queries.ReadTransfer(ctx, id)
	if err == sql.ErrNoRows {
		return transfer, ErrNotFound
	}
	if err != nil {
		return transfer, err
	}

	transfer.ID = row.Transferuuid
	transfer.Name = row.Description
	transfer.CurrentPath = row.Currentlocation

	// TODO: convert types.
	switch row.Type {
	case "standard":
		transfer.Type = adminv1.TransferType_TRANSFER_TYPE_STANDARD
	}

	switch row.Status {
	case uint16(enums.PackageStatusUnknown):
		transfer.Status = adminv1.PackageStatus_PACKAGE_STATUS_UNSPECIFIED
	case uint16(enums.PackageStatusProcessing):
		transfer.Status = adminv1.PackageStatus_PACKAGE_STATUS_PROCESSING
	case uint16(enums.PackageStatusDone):
		transfer.Status = adminv1.PackageStatus_PACKAGE_STATUS_DONE
	case uint16(enums.PackageStatusCompletedSuccessfully):
		transfer.Status = adminv1.PackageStatus_PACKAGE_STATUS_COMPLETED_SUCCESSFULLY
	case uint16(enums.PackageStatusFailed):
		transfer.Status = adminv1.PackageStatus_PACKAGE_STATUS_FAILED
	}

	return transfer, nil
}

func (s *mysqlStoreImpl) UpsertTransfer(ctx context.Context, id uuid.UUID, path string) (_ bool, err error) {
	defer wrap(&err, "UpsertTransfer(%s, %s)", id, path)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	r, err := q.ReadTransferLocation(ctx, id)

	// Return an error as we've failed to read the transfer.
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("read transfer: %v", err)
	}

	// Create the transfer as it has not been created yet.
	if err == sql.ErrNoRows {
		if err := q.CreateTransfer(ctx, &sqlc.CreateTransferParams{
			Transferuuid:    id,
			Currentlocation: path,
		}); err != nil {
			return false, fmt.Errorf("create transfer: %v", err)
		} else {
			return true, tx.Commit()
		}
	}

	// Update current location if needed.
	if r.Currentlocation == path {
		return false, nil
	}
	if err := q.UpdateTransferLocation(ctx, &sqlc.UpdateTransferLocationParams{
		Transferuuid:    id,
		Currentlocation: path,
	}); err != nil {
		return false, fmt.Errorf("update transfer: %v", err)
	}

	return false, tx.Commit()
}

func (s *mysqlStoreImpl) EnsureTransfer(ctx context.Context, path string) (_ uuid.UUID, _ bool, err error) {
	defer wrap(&err, "EnsureTransfer(%s)", path)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return uuid.Nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	id, err := q.ReadTransferWithLocation(ctx, path)

	// Return an error as we've failed to read the transfer.
	if err != nil && err != sql.ErrNoRows {
		return uuid.Nil, false, fmt.Errorf("read transfer: %v", err)
	}

	// Create the transfer as it has not been created yet.
	if err == sql.ErrNoRows {
		id := uuid.New()
		if err := q.CreateTransfer(ctx, &sqlc.CreateTransferParams{
			Transferuuid:    id,
			Currentlocation: path,
		}); err != nil {
			return uuid.Nil, false, fmt.Errorf("create transfer: %v", err)
		} else {
			return id, true, tx.Commit()
		}
	}

	return id, false, nil // Transfer found!
}

func (s *mysqlStoreImpl) UpdateTransferLocation(ctx context.Context, id uuid.UUID, path string) (err error) {
	defer wrap(&err, "UpdateTransferLocation(%s, %s)", id, path)

	return s.queries.UpdateTransferLocation(ctx, &sqlc.UpdateTransferLocationParams{
		Transferuuid:    id,
		Currentlocation: path,
	})
}

func (s *mysqlStoreImpl) ReadSIP(ctx context.Context, id uuid.UUID) (_ SIP, err error) {
	defer wrap(&err, "ReadSIP(%s)", id)

	sip := SIP{}

	row, err := s.queries.ReadSIP(ctx, id)
	if err == sql.ErrNoRows {
		return sip, ErrNotFound
	}
	if err != nil {
		return sip, err
	}

	sip.ID = row.SIPID
	sip.CreatedAt = row.CreatedAt
	sip.CurrentPath = row.Currentpath.String
	sip.Hidden = row.Hidden
	sip.AIPFilename = row.Aipfilename.String
	sip.DirIDs = row.Diruuids
	sip.Status = int(row.Status)
	sip.CompletedAt = row.CompletedAt.Time

	// SIP, AIC, AIP-REIN, AIC-REIN
	sip.Type = row.Siptype

	return sip, nil
}

func (s *mysqlStoreImpl) UpsertSIP(ctx context.Context, id uuid.UUID, path string) (_ bool, err error) {
	defer wrap(&err, "UpsertSIP(%s, %s)", id, path)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	r, err := q.ReadSIPLocation(ctx, id)

	// Return an error as we've failed to read the transfer.
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("read SIP: %v", err)
	}

	// Create the transfer as it has not been created yet.
	if err == sql.ErrNoRows {
		if err := q.CreateSIP(ctx, &sqlc.CreateSIPParams{
			SIPID:       id,
			Currentpath: sql.NullString{String: path, Valid: true},
			Siptype:     "SIP",
		}); err != nil {
			return false, fmt.Errorf("create SIP: %v", err)
		} else {
			return true, tx.Commit()
		}
	}

	// Update current location if needed.
	if r.Currentpath.String == path {
		return false, nil
	}
	if err := q.UpdateSIPLocation(ctx, &sqlc.UpdateSIPLocationParams{
		SIPID:       id,
		Currentpath: sql.NullString{String: path, Valid: true},
	}); err != nil {
		return false, fmt.Errorf("update SIP: %v", err)
	}

	return false, tx.Commit()
}

func (s *mysqlStoreImpl) EnsureSIP(ctx context.Context, path string) (_ uuid.UUID, _ bool, err error) {
	defer wrap(&err, "EnsureSIP(%s)", path)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return uuid.Nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	id, err := q.ReadSIPWithLocation(ctx, sql.NullString{String: path, Valid: true})

	// Return an error as we've failed to read the transfer.
	if err != nil && err != sql.ErrNoRows {
		return uuid.Nil, false, fmt.Errorf("read SIP: %v", err)
	}

	// Create the SIP as it has not been created yet.
	if err == sql.ErrNoRows {
		id := uuid.New()
		if err := q.CreateSIP(ctx, &sqlc.CreateSIPParams{
			SIPID:       id,
			Currentpath: sql.NullString{String: path, Valid: true},
			Siptype:     "SIP",
		}); err != nil {
			return uuid.Nil, false, fmt.Errorf("create SIP: %v", err)
		} else {
			return id, true, tx.Commit()
		}
	}

	return id, false, nil // SIP found!
}

func (s *mysqlStoreImpl) ReadDIP(ctx context.Context, id uuid.UUID) (dip DIP, err error) {
	defer wrap(&err, "ReadDIP(%s)", id)

	row, err := s.queries.ReadSIP(ctx, id)
	if err == sql.ErrNoRows {
		return dip, ErrNotFound
	}
	if err != nil {
		return dip, err
	}

	dip.ID = row.SIPID
	dip.CreatedAt = row.CreatedAt
	dip.CurrentPath = row.Currentpath.String
	dip.Hidden = row.Hidden
	dip.AIPFilename = row.Aipfilename.String
	dip.DirIDs = row.Diruuids
	dip.Status = int(row.Status)
	dip.CompletedAt = row.CompletedAt.Time

	return dip, nil
}

func (s *mysqlStoreImpl) UpsertDIP(ctx context.Context, id uuid.UUID, path string) (_ bool, err error) {
	defer wrap(&err, "UpsertDIP(%s, %s)", id, path)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	_, err = q.ReadSIPLocation(ctx, id)

	// Return an error as we've failed to read the transfer.
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("read DIP: %v", err)
	}

	// Create the transfer as it has not been created yet.
	if err == sql.ErrNoRows {
		if err := q.CreateSIP(ctx, &sqlc.CreateSIPParams{
			SIPID:       id,
			Currentpath: sql.NullString{String: path, Valid: true},
			Siptype:     "DIP",
		}); err != nil {
			return false, fmt.Errorf("create DIP: %v", err)
		} else {
			return true, tx.Commit()
		}
	}

	return false, tx.Commit()
}

func (s *mysqlStoreImpl) EnsureDIP(ctx context.Context, path string) (_ uuid.UUID, _ bool, err error) {
	defer wrap(&err, "EnsureDIP(%s)", path)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return uuid.Nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	id, err := q.ReadSIPWithLocation(ctx, sql.NullString{String: path, Valid: true})

	// Return an error as we've failed to read the transfer.
	if err != nil && err != sql.ErrNoRows {
		return uuid.Nil, false, fmt.Errorf("read DIP: %v", err)
	}

	// Create the SIP as it has not been created yet.
	if err == sql.ErrNoRows {
		id := uuid.New()
		if err := q.CreateSIP(ctx, &sqlc.CreateSIPParams{
			SIPID:       id,
			Currentpath: sql.NullString{String: path, Valid: true},
			Siptype:     "DIP",
		}); err != nil {
			return uuid.Nil, false, fmt.Errorf("create DIP: %v", err)
		} else {
			return id, true, tx.Commit()
		}
	}

	return id, false, nil // SIP found!
}

func (s *mysqlStoreImpl) ReadUnitVars(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name string) (vars []UnitVar, err error) {
	defer wrap(&err, "ReadUnitVars(%s, %s)", packageType, name)

	ret, err := s.queries.ReadUnitVars(ctx, &sqlc.ReadUnitVarsParams{
		UnitID: id,
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	for _, item := range ret {
		if packageType != "" && packageType.String() != item.Unittype.String {
			continue // Filter by package type if requested.
		}
		uv := UnitVar{}
		if item.Variablevalue.Valid {
			uv.Value = &item.Variablevalue.String
		}
		if item.LinkID.Valid {
			uv.LinkID = &item.LinkID.UUID
		}
		vars = append(vars, uv)
	}

	return vars, nil
}

func (s *mysqlStoreImpl) ReadUnitVar(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name string) (_ string, err error) {
	defer wrap(&err, "ReadUnitVar(%s, %s, %s)", id, packageType, name)

	ret, err := s.queries.ReadUnitVar(ctx, &sqlc.ReadUnitVarParams{
		UnitID: id,
		UnitType: sql.NullString{
			String: packageType.String(),
			Valid:  true,
		},
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return ret.Variablevalue.String, nil
}

func (s *mysqlStoreImpl) ReadUnitLinkID(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name string) (_ uuid.UUID, err error) {
	defer wrap(&err, "ReadUnitVarLinkID(%s, %s, %s)", id, packageType, name)

	ret, err := s.queries.ReadUnitVar(ctx, &sqlc.ReadUnitVarParams{
		UnitID: id,
		UnitType: sql.NullString{
			String: packageType.String(),
			Valid:  true,
		},
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	if err == sql.ErrNoRows {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	return ret.LinkID.UUID, nil
}

func (s *mysqlStoreImpl) CreateUnitVar(ctx context.Context, id uuid.UUID, packageType enums.PackageType, name, value string, linkID uuid.UUID, updateExisting bool) (err error) {
	defer wrap(&err, "CreateUnitVar(%s, %s, %s, %s)", id, packageType, name, value)

	tx, err := s.pool.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)

	exists := false
	uv, err := q.ReadUnitVar(ctx, &sqlc.ReadUnitVarParams{
		UnitID: id,
		UnitType: sql.NullString{
			String: packageType.String(),
			Valid:  true,
		},
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return err
	default:
		exists = true
	}

	var (
		wantValue  sql.NullString
		wantLinkID uuid.NullUUID
	)
	{
		switch {
		case value == "" && linkID == uuid.Nil:
			return errors.New("both value and linkID are zero")
		case value != "" && linkID != uuid.Nil:
			return errors.New("both value and linkID are non-zero")
		case value != "":
			// MCPServer sets "link_id" to NULL when a "value" is given, e.g.:
			// 	name="processingConfiguration", value="automated", link_id=NULL
			wantValue.String = value
			wantValue.Valid = true
			wantLinkID.Valid = false
		case linkID != uuid.Nil:
			// MCPServer sets "value" to empty string when a "link_id" is given, e.g.:
			//	name="reNormalize", value="", link_id="8ba83807-2832-4e41-843c-2e55ad10ea0b"/
			wantValue.Valid = true
			wantLinkID.UUID = linkID
			wantLinkID.Valid = true
		}
	}

	// It exists but it does not require further updates.
	if exists && wantValue == uv.Variablevalue && wantLinkID == uv.LinkID {
		return nil
	}

	// It exists and requires further updates but we rather raise an error.
	if !updateExisting {
		return errors.New("variable exists but with different propreties")
	}

	if exists {
		err := q.UpdateUnitVar(ctx, &sqlc.UpdateUnitVarParams{
			Value:  wantValue,
			LinkID: wantLinkID,
			// Where...
			UnitID:   id,
			UnitType: sql.NullString{String: packageType.String(), Valid: true},
			Name:     sql.NullString{String: name, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("update: %v", err)
		} else {
			return tx.Commit()
		}
	} else {
		if err := s.queries.CreateUnitVar(ctx, &sqlc.CreateUnitVarParams{
			UnitID: id,
			UnitType: sql.NullString{
				String: packageType.String(),
				Valid:  true,
			},
			Name: sql.NullString{
				String: name,
				Valid:  true,
			},
			Value:  wantValue,
			LinkID: wantLinkID,
		}); err != nil {
			return fmt.Errorf("create: %v", err)
		} else {
			return tx.Commit()
		}
	}
}

func (s *mysqlStoreImpl) Files(ctx context.Context, id uuid.UUID, packageType enums.PackageType, filterFilenameEnd, filterSubdir, replacementPath string) (_ []File, err error) {
	defer wrap(&err, "Files(%s, %s, %s, %s, %s)", id, packageType, filterFilenameEnd, filterSubdir, replacementPath)

	sel := s.goqu.Select().From(myFilesTable)
	if filterFilenameEnd != "" {
		sel = sel.Where(goqu.Ex{"currentLocation": goqu.Op{"like": "%" + filterFilenameEnd}})
	}
	if filterSubdir != "" {
		sel = sel.Where(goqu.Ex{"currentLocation": goqu.Op{"like": replacementPath + filterSubdir + "%"}})
	}
	switch packageType {
	case enums.PackageTypeTransfer:
		sel = sel.Where(goqu.Ex{"transferUUID": id.String()})
	case enums.PackageTypeSIP, enums.PackageTypeDIP:
		sel = sel.Where(goqu.Ex{"sipUUID": id.String()})
	default:
		return nil, fmt.Errorf("unexpected package type: %q", packageType)
	}

	ret := []File{}

	const batchSize = 250
	offset := uint(0)
	for {
		sel = sel.Limit(batchSize).Offset(offset)
		files := make([]File, 0, batchSize)
		if err := sel.ScanStructsContext(ctx, &files); err != nil {
			return nil, fmt.Errorf("scan structs: %v", err)
		}
		if len(files) == 0 {
			break
		} else {
			ret = append(ret, files...)
			offset += uint(len(files))
		}
	}

	return ret, nil
}

func (s *mysqlStoreImpl) ReadPipelineID(ctx context.Context) (_ uuid.UUID, err error) {
	defer wrap(&err, "ReadPipelineID()")

	ret, err := s.queries.ReadDashboardSetting(ctx, "dashboard_uuid")
	if err == sql.ErrNoRows {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.Parse(ret.Value)
	if err != nil {
		return uuid.Nil, err
	}

	return id, err
}

func (s *mysqlStoreImpl) ReadDict(ctx context.Context, name string) (_ map[string]string, err error) {
	defer wrap(&err, "ReadDict(%s)", name)

	rows, err := s.queries.ReadDashboardSettingsWithScope(ctx, name)
	if err != nil {
		return nil, err
	}
	ln := len(rows)
	if ln == 0 {
		return nil, ErrNotFound
	}

	ret := make(map[string]string, ln)
	for _, row := range rows {
		ret[row.Name] = ret[row.Value]
	}

	return ret, nil
}

func (s *mysqlStoreImpl) ValidateUserAPIKey(ctx context.Context, username, key string) (_ *User, err error) {
	defer wrap(&err, "ValidateUserAPIKey(%q, %q)", username, key)

	row, err := s.queries.ReadUserWithKey(ctx, &sqlc.ReadUserWithKeyParams{
		Username: username,
		Key:      key,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	ret := &User{
		ID:       int(row.ID),
		Username: row.Username,
		Email:    row.Email,
		Active:   row.IsActive,
	}
	if row.AgentID.Valid {
		ret.AgentID = ref.New(int(row.AgentID.Int32))
	}

	return ret, nil
}

func (s *mysqlStoreImpl) Running() bool {
	return s != nil
}

func (s *mysqlStoreImpl) Close() error {
	var err error

	if s.pool != nil {
		err = errors.Join(err, s.pool.Close())
	}

	if s.queries != nil {
		err = errors.Join(err, s.queries.Close())
	}

	return err
}

func wrap(errp *error, format string, args ...any) {
	if *errp == nil {
		return
	}
	var (
		errfmt  string
		message = fmt.Sprintf(format, args...)
	)
	if *errp == ErrNotFound {
		errfmt = "%s: %w"
	} else {
		errfmt = "%s: %v"
	}
	*errp = fmt.Errorf(errfmt, message, *errp)
}

func updateTimeWithFraction(dst **timestamppb.Timestamp, t time.Time, dec string) error {
	df, err := strconv.ParseFloat(dec, 64)
	if err != nil {
		return fmt.Errorf("error parsing fractional seconds: %v", err)
	}

	ns := int64(df * float64(time.Second))
	t = t.Add(time.Duration(ns))

	*dst = timestamppb.New(t)

	return nil
}
