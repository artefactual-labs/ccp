package controller

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/artefactual/archivematica/hack/ccp/internal/derrors"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
)

// This file contains functions that relate to the process of retrieving
// transfers from transfer source locations that may or may not be provided by
// Archivematica Storage Service. This is built after what we have in the
// `server.packages` Python module.

// StartTransfer starts a transfer.
//
// The transfer is deposited into the internal processing directory and the
// workflow is triggered manually.
//
// This method does not rely on the activeTransfer watched directory. It does
// not prompt the user to accept the transfer because we go directly into the
// next chain link.
func StartTransfer(sharedDir, tmpDir, name, path string) error {
	destRel, destAbs, src := determineTransferPaths(sharedDir, tmpDir, name, path)
	fmt.Println(destRel, destAbs, src)

	copyFromTransferSources(nil)

	tsrc, tdst := "", ""
	dst, err := moveToInternalSharedDir(sharedDir, tsrc, tdst)
	if err != nil {
		return err
	}

	fmt.Println("The transfer is now in the internal processing directory!", dst)

	// TODO: update transfer.currentlocation in the database.
	// TODO: schedule job chain.

	return nil
}

// StartTransferWithWatchedDir starts a transfer using watched directories.
//
// This means copying the transfer into one of the standard watched dirs.
// MCPServer will continue the processing and prompt the user once the
// contents in the watched directory are detected by the watched directory
// observer.
//
// With this method of starting a transfer, the workflow requires user approval.
// This allows for adding metadata to the transfer before accepting it.
func StartTransferWithWatchedDir() {
	panic("not implemented")
	// _determine_transfer_paths
	// _copy_from_transfer_sources
	// _move_to_internal_shared_dir
	// update transfer.currentlocation with the new destination
}

func locationPath(locPath string) (id, path string) {
	if before, after, found := strings.Cut(locPath, ":"); found {
		id = before
		path = after
	} else {
		id = before
	}

	return id, path
}

// determineTransferPaths
//
// name and path are part of the client transfer request.
func determineTransferPaths(sharedDir, tmpDir, name, path string) (string, string, string) {
	archived := false
	lpath := strings.ToLower(path)
	if strings.HasSuffix(lpath, ".zip") ||
		strings.HasSuffix(lpath, ".tgz") ||
		strings.HasSuffix(lpath, ".tar.gz") {
		archived = true
	}

	var (
		transferDir string
		destAbs     string
	)

	if archived {
		transferDir = tmpDir
		_, p := locationPath(path)
		destAbs = filepath.Join(tmpDir, filepath.Base(p))
	} else {
		path = joinPath(path, "") // Copy contents of dir but not dir.
		destAbs = filepath.Join(tmpDir, name)
		transferDir = destAbs
	}

	destRel := strings.Replace(transferDir, sharedDir, "", 1)

	return destRel, destAbs, path
}

// moveToInternalSharedDir moves a transfer into an internal directory.
func moveToInternalSharedDir(sharedDir, path, dest string) (_ string, err error) {
	defer derrors.Add(&err, "moveToInternalSharedDir(%s, %s, %s)", sharedDir, path, dest)

	// Validate path.
	if path == "" {
		return "", errors.New("no path provided")
	}
	if strings.Contains(path, "..") {
		return "", errors.New("illegal path")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", errors.New("path does not exist")
	}

	var (
		attempt   = 0
		suggested = filepath.Join(dest, filepath.Base(path))
		newPath   = suggested
	)
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if err := os.Rename(path, newPath); os.IsExist(err) {
				goto incr // Magic!
			} else if err != nil {
				return "", err
			}

			return newPath, nil // Success!
		}

	incr:
		attempt++
		if attempt > 1000 {
			return "", fmt.Errorf("reached max. number of attempts: %d", attempt)
		}

		ext := filepath.Ext(dest)
		base := strings.TrimSuffix(suggested, ext)
		newPath = fmt.Sprintf("%s_%d%s", base, attempt, ext)
	}
}

func copyFromTransferSources(c ssclient.Client) {
	// - processing_location = storage_service.get_first_location(purpose="CP")
	// - transfer_sources = storage_service.get_location(purpose="TS")
	// - _default_transfer_source_location_uuid
	// - storage_service.copy_files(location, processing_location, files)
}
