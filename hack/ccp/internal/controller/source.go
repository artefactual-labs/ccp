package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/artefactual/archivematica/hack/ccp/internal/derrors"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/enums"
)

// This file contains functions that relate to the process of retrieving
// transfers from transfer source locations that may or may not be provided by
// Archivematica Storage Service. This is built after what we have in the
// `server.packages` Python module.

// copyTransfer copies the contents of a transfer into the processing directory.
//
// The transfer is deposited into the internal processing directory and the
// workflow is triggered manually.
//
// This method does not rely on the activeTransfer watched directory. It does
// not prompt the user to accept the transfer because we go directly into the
// next chain link.
func copyTransfer(ctx context.Context, ssclient ssclient.Client, sharedDir, tmpDir, name, path string) (string, error) {
	destRel, destAbs, _ := determineTransferPaths(sharedDir, tmpDir, name, path)

	if err := copyFromTransferSources(ctx, ssclient, sharedDir, []string{path}, destRel); err != nil {
		return "", err
	}

	final, err := moveToInternalSharedDir(
		destAbs,
		filepath.Join(joinPath(sharedDir, "currentlyProcessing")),
	)

	return final, err
}

// copyTransferIntoActiveTransfers starts a transfer using watched directories.
//
// This means copying the transfer into one of the standard watched dirs.
// MCPServer will continue the processing and prompt the user once the
// contents in the watched directory are detected by the watched directory
// observer.
//
// With this method of starting a transfer, the workflow requires user approval.
// This allows for adding metadata to the transfer before accepting it.
func copyTransferIntoActiveTransfers() { // nolint: unused
	panic("not implemented")
	// _determine_transfer_paths
	// _copy_from_transfer_sources
	// _move_to_internal_shared_dir
	// update transfer.currentlocation with the new destination
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
		path = joinPath(path, ".") // Copy contents of dir but not dir.
		destAbs = filepath.Join(tmpDir, name)
		transferDir = destAbs
	}

	destRel := strings.Replace(transferDir, sharedDir, "", 1)

	return destRel, destAbs, path
}

// moveToInternalSharedDir moves a transfer into an internal directory.
func moveToInternalSharedDir(src, dst string) (_ string, err error) {
	defer derrors.Add(&err, "moveToInternalSharedDir(%s, %s)", src, dst)

	// Validate path.
	if src == "" {
		return "", errors.New("no path provided")
	}
	if strings.Contains(src, "..") {
		return "", fmt.Errorf("illegal path: %q", src)
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %q", src)
	}

	var (
		attempt   = 0
		suggested = filepath.Join(dst, filepath.Base(src))
		newPath   = suggested
	)
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if err := os.Rename(src, newPath); os.IsExist(err) {
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

		ext := filepath.Ext(dst)
		base := strings.TrimSuffix(suggested, ext)
		newPath = fmt.Sprintf("%s_%d%s", base, attempt, ext)
	}
}

func copyFromTransferSources(ctx context.Context, c ssclient.Client, sharedDir string, paths []string, destRel string) (err error) {
	derrors.Add(&err, "copyFromTransferSources()")

	// We'll use the default transfer source location when a request does not
	// indicate its source.
	defaultTransferSource, err := c.ReadDefaultLocation(ctx, enums.LocationPurposeTS)
	if err != nil {
		return err
	}

	// Look up the destination, which is our pipeline processing location.
	currentlyProcessing, err := c.ReadProcessingLocation(ctx)
	if err != nil {
		return err
	}

	// filesByLocID is a list of all the copy operations that we'll be making,
	// indexed by the identifier of the transfer source location.
	transferSources, err := c.ListLocations(ctx, "", enums.LocationPurposeTS)
	if err != nil {
		return err
	}
	type sourceFiles struct {
		transferSource *ssclient.Location
		files          [][2]string // src, dst
	}
	filesByLocID := map[uuid.UUID]*sourceFiles{}
	for _, loc := range transferSources {
		filesByLocID[loc.ID] = &sourceFiles{
			transferSource: loc,
			files:          [][2]string{},
		}
	}

	for _, item := range paths {
		locID, path := locationPath(item)
		if locID == uuid.Nil {
			locID = defaultTransferSource.ID
		}
		ops, ok := filesByLocID[locID]
		if !ok {
			return fmt.Errorf("location %s is not associated with this pipeline", locID)
		}

		dir := isDir(filepath.Join(sharedDir, "tmp", strings.TrimPrefix("/", destRel)))
		fmt.Println(dir, sharedDir)

		// Source relative to the transfer source path.
		source := strings.Replace(path, ops.transferSource.Path, "", 1)
		source = strings.TrimPrefix(source, "/")

		// # Use the last segment of the path for the destination - basename for
		// # a file, or the last folder if not. Keep the trailing / for folders.
		//
		// TODO: this is broken.
		/*
			var lastSegment string
			if dir {
				lastSegment = joinPath(filepath.Base(filepath.Dir(source)), "")
			} else {
				lastSegment = filepath.Base(source)
			}
		*/

		destination := joinPath(currentlyProcessing.Path, destRel, "") + "."
		destination = strings.Replace(destination, "%sharedPath%", "", 1)

		// What SS expects must look like this:
		//
		//	[{
		//		'source': 'archivematica/transfer/.',
		//		'destination': '/var/archivematica/sharedDirectory/tmp/tmp9an4_1zv/20240521104109/.'
		//	}]
		//
		//	[{
		//		'source': 'archivematica/archivematica-sampledata/SampleTransfers/Images/pictures/.',
		//		'destination': '/var/archivematica/sharedDirectory/tmp/tmpzwq0mg0r/20240521104254/.'
		//	}]
		ops.files = append(ops.files, [2]string{source, destination})
	}

	for _, sf := range filesByLocID {
		if copyErr := c.MoveFiles(ctx, sf.transferSource, currentlyProcessing, sf.files); copyErr != nil {
			err = errors.Join(err, copyErr)
		}
	}

	return err
}
