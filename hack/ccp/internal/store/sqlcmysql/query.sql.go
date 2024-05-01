// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: query.sql

package sqlcmysql

import (
	"context"
	"database/sql"
	"time"

	uuid "github.com/google/uuid"
)

const cleanUpActiveJobs = `-- name: CleanUpActiveJobs :exec
UPDATE Jobs SET currentStep = 4 WHERE currentStep = 3
`

func (q *Queries) CleanUpActiveJobs(ctx context.Context) error {
	_, err := q.exec(ctx, q.cleanUpActiveJobsStmt, cleanUpActiveJobs)
	return err
}

const cleanUpActiveSIPs = `-- name: CleanUpActiveSIPs :exec
UPDATE SIPs SET status = 4, completed_at = UTC_TIMESTAMP() WHERE status IN (0, 1)
`

func (q *Queries) CleanUpActiveSIPs(ctx context.Context) error {
	_, err := q.exec(ctx, q.cleanUpActiveSIPsStmt, cleanUpActiveSIPs)
	return err
}

const cleanUpActiveTasks = `-- name: CleanUpActiveTasks :exec
UPDATE Tasks SET exitCode = -1, stdError = "MCP shut down while processing." WHERE exitCode IS NULL
`

func (q *Queries) CleanUpActiveTasks(ctx context.Context) error {
	_, err := q.exec(ctx, q.cleanUpActiveTasksStmt, cleanUpActiveTasks)
	return err
}

const cleanUpActiveTransfers = `-- name: CleanUpActiveTransfers :exec
UPDATE Transfers SET status = 4, completed_at = UTC_TIMESTAMP() WHERE status IN (0, 1)
`

func (q *Queries) CleanUpActiveTransfers(ctx context.Context) error {
	_, err := q.exec(ctx, q.cleanUpActiveTransfersStmt, cleanUpActiveTransfers)
	return err
}

const cleanUpAwaitingJobs = `-- name: CleanUpAwaitingJobs :exec
DELETE FROM Jobs WHERE currentStep = 1
`

func (q *Queries) CleanUpAwaitingJobs(ctx context.Context) error {
	_, err := q.exec(ctx, q.cleanUpAwaitingJobsStmt, cleanUpAwaitingJobs)
	return err
}

const cleanUpTasksWithAwaitingJobs = `-- name: CleanUpTasksWithAwaitingJobs :exec
DELETE FROM Tasks WHERE jobuuid IN (SELECT jobUUID FROM Jobs WHERE currentStep = 1)
`

func (q *Queries) CleanUpTasksWithAwaitingJobs(ctx context.Context) error {
	_, err := q.exec(ctx, q.cleanUpTasksWithAwaitingJobsStmt, cleanUpTasksWithAwaitingJobs)
	return err
}

const createJob = `-- name: CreateJob :exec
INSERT INTO Jobs (jobUUID, jobType, createdTime, createdTimeDec, directory, SIPUUID, unitType, currentStep, microserviceGroup, hidden, MicroServiceChainLinksPK, subJobOf) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type CreateJobParams struct {
	ID                       uuid.UUID
	Type                     string
	CreatedAt                time.Time
	Createdtimedec           string
	Directory                string
	SIPID                    uuid.UUID
	Unittype                 string
	Currentstep              int32
	Microservicegroup        string
	Hidden                   bool
	Microservicechainlinkspk sql.NullString
	Subjobof                 string
}

func (q *Queries) CreateJob(ctx context.Context, arg *CreateJobParams) error {
	_, err := q.exec(ctx, q.createJobStmt, createJob,
		arg.ID,
		arg.Type,
		arg.CreatedAt,
		arg.Createdtimedec,
		arg.Directory,
		arg.SIPID,
		arg.Unittype,
		arg.Currentstep,
		arg.Microservicegroup,
		arg.Hidden,
		arg.Microservicechainlinkspk,
		arg.Subjobof,
	)
	return err
}

const createTransfer = `-- name: CreateTransfer :exec
INSERT INTO Transfers (transferUUID, currentLocation, type, accessionID, sourceOfAcquisition, typeOfTransfer, description, notes, access_system_id, hidden, transferMetadataSetRowUUID, dirUUIDs, status, completed_at) VALUES (?, ?, '', '', '', '', '', '', '', 0, NULL, 0, 0, NULL)
`

type CreateTransferParams struct {
	Transferuuid    uuid.UUID
	Currentlocation string
}

func (q *Queries) CreateTransfer(ctx context.Context, arg *CreateTransferParams) error {
	_, err := q.exec(ctx, q.createTransferStmt, createTransfer, arg.Transferuuid, arg.Currentlocation)
	return err
}

const createUnitVar = `-- name: CreateUnitVar :exec
INSERT INTO UnitVariables (pk, unitType, unitUUID, variable, variableValue, microServiceChainLink, createdTime, updatedTime)
VALUES (
    UUID(),
    ?,
    ?,
    ?,
    ?,
    ?,
    UTC_TIMESTAMP(),
    UTC_TIMESTAMP()
)
`

type CreateUnitVarParams struct {
	UnitType sql.NullString
	UnitID   uuid.UUID
	Name     sql.NullString
	Value    sql.NullString
	LinkID   uuid.NullUUID
}

func (q *Queries) CreateUnitVar(ctx context.Context, arg *CreateUnitVarParams) error {
	_, err := q.exec(ctx, q.createUnitVarStmt, createUnitVar,
		arg.UnitType,
		arg.UnitID,
		arg.Name,
		arg.Value,
		arg.LinkID,
	)
	return err
}

const getLock = `-- name: GetLock :one
SELECT COALESCE(GET_LOCK('lock', 0), 0)
`

func (q *Queries) GetLock(ctx context.Context) (interface{}, error) {
	row := q.queryRow(ctx, q.getLockStmt, getLock)
	var coalesce interface{}
	err := row.Scan(&coalesce)
	return coalesce, err
}

const readDashboardSetting = `-- name: ReadDashboardSetting :one
SELECT name, value, scope FROM DashboardSettings WHERE name = ?
`

type ReadDashboardSettingRow struct {
	Name  string
	Value string
	Scope string
}

func (q *Queries) ReadDashboardSetting(ctx context.Context, name string) (*ReadDashboardSettingRow, error) {
	row := q.queryRow(ctx, q.readDashboardSettingStmt, readDashboardSetting, name)
	var i ReadDashboardSettingRow
	err := row.Scan(&i.Name, &i.Value, &i.Scope)
	return &i, err
}

const readDashboardSettingsWithNameLike = `-- name: ReadDashboardSettingsWithNameLike :many
SELECT name, value, scope FROM DashboardSettings WHERE name LIKE ?
`

type ReadDashboardSettingsWithNameLikeRow struct {
	Name  string
	Value string
	Scope string
}

func (q *Queries) ReadDashboardSettingsWithNameLike(ctx context.Context, name string) ([]*ReadDashboardSettingsWithNameLikeRow, error) {
	rows, err := q.query(ctx, q.readDashboardSettingsWithNameLikeStmt, readDashboardSettingsWithNameLike, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*ReadDashboardSettingsWithNameLikeRow{}
	for rows.Next() {
		var i ReadDashboardSettingsWithNameLikeRow
		if err := rows.Scan(&i.Name, &i.Value, &i.Scope); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const readDashboardSettingsWithScope = `-- name: ReadDashboardSettingsWithScope :many
SELECT name, value, scope FROM DashboardSettings WHERE scope = ?
`

type ReadDashboardSettingsWithScopeRow struct {
	Name  string
	Value string
	Scope string
}

func (q *Queries) ReadDashboardSettingsWithScope(ctx context.Context, scope string) ([]*ReadDashboardSettingsWithScopeRow, error) {
	rows, err := q.query(ctx, q.readDashboardSettingsWithScopeStmt, readDashboardSettingsWithScope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*ReadDashboardSettingsWithScopeRow{}
	for rows.Next() {
		var i ReadDashboardSettingsWithScopeRow
		if err := rows.Scan(&i.Name, &i.Value, &i.Scope); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const readTransferLocation = `-- name: ReadTransferLocation :one
SELECT transferUUID, currentLocation FROM Transfers WHERE transferUUID = ?
`

type ReadTransferLocationRow struct {
	Transferuuid    uuid.UUID
	Currentlocation string
}

func (q *Queries) ReadTransferLocation(ctx context.Context, transferuuid uuid.UUID) (*ReadTransferLocationRow, error) {
	row := q.queryRow(ctx, q.readTransferLocationStmt, readTransferLocation, transferuuid)
	var i ReadTransferLocationRow
	err := row.Scan(&i.Transferuuid, &i.Currentlocation)
	return &i, err
}

const readTransferWithLocation = `-- name: ReadTransferWithLocation :one
SELECT transferUUID FROM Transfers WHERE currentLocation = ?
`

func (q *Queries) ReadTransferWithLocation(ctx context.Context, currentlocation string) (uuid.UUID, error) {
	row := q.queryRow(ctx, q.readTransferWithLocationStmt, readTransferWithLocation, currentlocation)
	var transferuuid uuid.UUID
	err := row.Scan(&transferuuid)
	return transferuuid, err
}

const readUnitVar = `-- name: ReadUnitVar :one
SELECT variableValue, microServiceChainLink FROM UnitVariables WHERE unitType = ? AND unitUUID = ? AND variable = ?
`

type ReadUnitVarParams struct {
	UnitType sql.NullString
	UnitID   uuid.UUID
	Name     sql.NullString
}

type ReadUnitVarRow struct {
	Variablevalue sql.NullString
	LinkID        uuid.NullUUID
}

func (q *Queries) ReadUnitVar(ctx context.Context, arg *ReadUnitVarParams) (*ReadUnitVarRow, error) {
	row := q.queryRow(ctx, q.readUnitVarStmt, readUnitVar, arg.UnitType, arg.UnitID, arg.Name)
	var i ReadUnitVarRow
	err := row.Scan(&i.Variablevalue, &i.LinkID)
	return &i, err
}

const readUnitVars = `-- name: ReadUnitVars :many
SELECT unitType, unitUUID, variable, variableValue, microServiceChainLink FROM UnitVariables WHERE unitUUID = ? AND variable = ?
`

type ReadUnitVarsParams struct {
	UnitID uuid.UUID
	Name   sql.NullString
}

type ReadUnitVarsRow struct {
	Unittype      sql.NullString
	Unituuid      uuid.UUID
	Variable      sql.NullString
	Variablevalue sql.NullString
	LinkID        uuid.NullUUID
}

func (q *Queries) ReadUnitVars(ctx context.Context, arg *ReadUnitVarsParams) ([]*ReadUnitVarsRow, error) {
	rows, err := q.query(ctx, q.readUnitVarsStmt, readUnitVars, arg.UnitID, arg.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*ReadUnitVarsRow{}
	for rows.Next() {
		var i ReadUnitVarsRow
		if err := rows.Scan(
			&i.Unittype,
			&i.Unituuid,
			&i.Variable,
			&i.Variablevalue,
			&i.LinkID,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const releaseLock = `-- name: ReleaseLock :one
SELECT RELEASE_LOCK('lock')
`

func (q *Queries) ReleaseLock(ctx context.Context) (bool, error) {
	row := q.queryRow(ctx, q.releaseLockStmt, releaseLock)
	var release_lock bool
	err := row.Scan(&release_lock)
	return release_lock, err
}

const updateJobStatus = `-- name: UpdateJobStatus :exec
UPDATE Jobs SET currentStep = ? WHERE jobUUID = ?
`

type UpdateJobStatusParams struct {
	Currentstep int32
	ID          uuid.UUID
}

func (q *Queries) UpdateJobStatus(ctx context.Context, arg *UpdateJobStatusParams) error {
	_, err := q.exec(ctx, q.updateJobStatusStmt, updateJobStatus, arg.Currentstep, arg.ID)
	return err
}

const updateTransferLocation = `-- name: UpdateTransferLocation :exec
UPDATE Transfers SET currentLocation = ? WHERE transferUUID = ?
`

type UpdateTransferLocationParams struct {
	Currentlocation string
	Transferuuid    uuid.UUID
}

func (q *Queries) UpdateTransferLocation(ctx context.Context, arg *UpdateTransferLocationParams) error {
	_, err := q.exec(ctx, q.updateTransferLocationStmt, updateTransferLocation, arg.Currentlocation, arg.Transferuuid)
	return err
}

const updateUnitVar = `-- name: UpdateUnitVar :exec
UPDATE UnitVariables
SET
    variableValue = ?,
    microServiceChainLink = ?,
    updatedTime = UTC_TIMESTAMP()
WHERE
    unitType = ?
    AND unitUUID = ?
    AND variable = ?
`

type UpdateUnitVarParams struct {
	Value    sql.NullString
	LinkID   uuid.NullUUID
	UnitType sql.NullString
	UnitID   uuid.UUID
	Name     sql.NullString
}

func (q *Queries) UpdateUnitVar(ctx context.Context, arg *UpdateUnitVarParams) error {
	_, err := q.exec(ctx, q.updateUnitVarStmt, updateUnitVar,
		arg.Value,
		arg.LinkID,
		arg.UnitType,
		arg.UnitID,
		arg.Name,
	)
	return err
}
