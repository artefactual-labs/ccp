--
-- Jobs
--

-- name: CreateJob :exec
INSERT INTO Jobs (jobUUID, jobType, createdTime, createdTimeDec, directory, SIPUUID, unitType, currentStep, microserviceGroup, hidden, MicroServiceChainLinksPK, subJobOf) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateJobStatus :exec
UPDATE Jobs SET currentStep = ? WHERE jobUUID = ?;

--
-- Transfers
--

-- name: CreateTransfer :exec
INSERT INTO Transfers (transferUUID, currentLocation, type, accessionID, sourceOfAcquisition, typeOfTransfer, description, notes, access_system_id, hidden, transferMetadataSetRowUUID, dirUUIDs, status, completed_at) VALUES (?, ?, '', '', '', '', '', '', '', 0, NULL, 0, 0, NULL);

-- name: ReadTransferLocation :one
SELECT transferUUID, currentLocation FROM Transfers WHERE transferUUID = ?;

-- name: ReadTransferWithLocation :one
SELECT transferUUID FROM Transfers WHERE currentLocation = ?;

-- name: UpdateTransferLocation :exec
UPDATE Transfers SET currentLocation = ? WHERE transferUUID = ?;

-- name: UpdateTransferStatus :exec
UPDATE Transfers SET status = ? WHERE transferUUID = ?;

--
-- SIPs
--

-- name: CreateSIP :exec
INSERT INTO SIPs (sipUUID, createdTime, currentPath, hidden, aipFilename, sipType, dirUUIDs, status, completed_at) VALUES (?, UTC_TIMESTAMP(), ?, 0, '', ?, 0, 0, NULL);

-- name: ReadSIP :one
SELECT sipUUID, createdTime, currentPath, hidden, aipFilename, sipType, dirUUIDs, status, completed_at FROM SIPs WHERE sipUUID = ?;

-- name: ReadSIPLocation :one
SELECT sipUUID, currentPath FROM SIPs WHERE sipUUID = ?;

-- name: ReadSIPWithLocation :one
SELECT sipUUID FROM SIPs WHERE currentPath = ?;

-- name: UpdateSIPLocation :exec
UPDATE SIPs SET currentPath = ? WHERE sipUUID = ?;

-- name: UpdateSIPStatus :exec
UPDATE SIPs SET status = ? WHERE sipUUID = ?;

--
-- Clean-ups
--

-- name: CleanUpTasksWithAwaitingJobs :exec
DELETE FROM Tasks WHERE jobuuid IN (SELECT jobUUID FROM Jobs WHERE currentStep = 1);

-- name: CleanUpAwaitingJobs :exec
DELETE FROM Jobs WHERE currentStep = 1;

-- name: CleanUpActiveJobs :exec
UPDATE Jobs SET currentStep = 4 WHERE currentStep = 3;

-- name: CleanUpActiveTransfers :exec
UPDATE Transfers SET status = 4, completed_at = UTC_TIMESTAMP() WHERE status IN (0, 1);

-- name: CleanUpActiveSIPs :exec
UPDATE SIPs SET status = 4, completed_at = UTC_TIMESTAMP() WHERE status IN (0, 1);

-- name: CleanUpActiveTasks :exec
UPDATE Tasks SET exitCode = -1, stdError = "MCP shut down while processing." WHERE exitCode IS NULL;

--
-- Unit variables
--

-- name: ReadUnitVar :one
SELECT variableValue, microServiceChainLink FROM UnitVariables WHERE unitType = sqlc.arg(unit_type) AND unitUUID = sqlc.arg(unit_id) AND variable = sqlc.arg(name);

-- name: ReadUnitVars :many
SELECT unitType, unitUUID, variable, variableValue, microServiceChainLink FROM UnitVariables WHERE unitUUID = sqlc.arg(unit_id) AND variable = sqlc.arg(name);

-- name: CreateUnitVar :exec
INSERT INTO UnitVariables (pk, unitType, unitUUID, variable, variableValue, microServiceChainLink, createdTime, updatedTime)
VALUES (
    UUID(),
    sqlc.arg(unit_type),
    sqlc.arg(unit_id),
    sqlc.arg(name),
    sqlc.arg(value),
    sqlc.arg(link_id),
    UTC_TIMESTAMP(),
    UTC_TIMESTAMP()
);


-- name: UpdateUnitVar :exec
UPDATE UnitVariables
SET
    variableValue = sqlc.arg(value),
    microServiceChainLink = sqlc.arg(link_id),
    updatedTime = UTC_TIMESTAMP()
WHERE
    unitType = sqlc.arg(unit_type)
    AND unitUUID = sqlc.arg(unit_id)
    AND variable = sqlc.arg(name);

--
-- Dashboard settings
--

-- name: ReadDashboardSettingsWithScope :many
SELECT name, value, scope FROM DashboardSettings WHERE scope = ?;

-- name: ReadDashboardSettingsWithNameLike :many
SELECT name, value, scope FROM DashboardSettings WHERE name LIKE ?;

-- name: ReadDashboardSetting :one
SELECT name, value, scope FROM DashboardSettings WHERE name = ?;
