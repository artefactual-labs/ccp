package controller

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/enums"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

// A Package can be a Transfer, a SIP, or a DIP.
type Package struct {
	logger logr.Logger

	// Datastore.
	store store.Store

	// Path of the shared directory.
	sharedDir string

	// The underlying package type.
	unit

	// Identifier, populated by hydrate().
	id uuid.UUID

	// Current path, populated by hydrate().
	path string

	// Identifier of the chain where the iterator must start processing.
	startAtChainID uuid.UUID

	// Identifier of the link where the iterator must start processing.
	startAtLinkID uuid.UUID
}

func newPackage(logger logr.Logger, store store.Store, sharedDir string) *Package {
	return &Package{
		logger:    logger,
		store:     store,
		sharedDir: joinPath(sharedDir, ""),
	}
}

// NewPackage creates a new package after a path observed in a watched directory.
func NewPackage(ctx context.Context, logger logr.Logger, store store.Store, sharedDir, path string, wd *workflow.WatchedDirectory) (*Package, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat: %v", err)
	}
	isDir := fi.IsDir()

	pkg := newPackage(logger, store, sharedDir)
	pkg.path = path
	pkg.startAtChainID = wd.ChainID

	switch {
	case wd.UnitType == "Transfer":
		pkg.unit = &Transfer{pkg: pkg}
	case wd.UnitType == "SIP" && isDir:
		pkg.unit = &SIP{pkg: pkg}
	case wd.UnitType == "DIP" && isDir:
		pkg.unit = &DIP{pkg: pkg}
	default:
		return nil, fmt.Errorf("unexpected type given for file %q (dir: %t)", path, isDir)
	}

	if err := pkg.hydrate(ctx, path, wd.Path); err != nil {
		return nil, fmt.Errorf("hydrate: %v", err)
	}

	return pkg, nil
}

// NewTransferPackage creates a new package after an API request.
//
//  1. Create Package (Transfer).
//     transfer = models.Transfer.objects.create(**kwargs)
//     if not processing_configuration_file_exists(processing_config): processing_config = "default"
//     transfer.set_processing_configuration(processing_config)
//     transfer.update_active_agent(user_id)
//
//  2. Create temporary directory inside sharedDir/tmp. [DONE]
//     tmpdir = mkdtemp(dir=os.path.join(_get_setting("SHARED_DIRECTORY"), "tmp"))
//
//  3. Identify starting point. [DONE]
//     starting_point = PACKAGE_TYPE_STARTING_POINTS.get(type_)
//
//  4. Start creation.
//     params = (transfer, name, path, tmpdir, starting_point)
//     if auto_approve:
//     params = params + (workflow, package_queue)
//     result = executor.submit(_start_package_transfer_with_auto_approval, *params)
//     else:
//     result = executor.submit(_start_package_transfer, *params)
//
//  5. Adjust permissions?
//     result.add_done_callback(lambda f: os.chmod(tmpdir, 0o770))
func NewTransferPackage(
	ctx context.Context,
	logger logr.Logger,
	store store.Store,
	ssclient ssclient.Client,
	sharedDir string,
	req *adminv1.CreatePackageRequest,
	queue func(pkg *Package),
) (*Package, error) {
	// TODO: implement transfer submissions without auto-approval (see: copyTransferIntoActiveTransfers).
	autoApprove := req.AutoApprove == nil || req.AutoApprove.Value
	if !autoApprove {
		return nil, errors.New("submissions with auto-approve disabled are not supported yet")
	}

	pkg := newPackage(logger, store, sharedDir)
	pkg.id = uuid.New()
	pkg.unit = &Transfer{pkg: pkg}

	transferType := Transfers.WithType(req.Type)
	pkg.startAtChainID = transferType.BypassChainID
	pkg.startAtLinkID = transferType.BypassLinkID

	var msID uuid.UUID
	if req.MetadataSetId != nil {
		msID, _ = uuid.Parse(req.MetadataSetId.Value)
	}

	err := store.CreateTransfer(ctx, pkg.id, req.Accession, req.AccessSystemId, msID)
	if err != nil {
		return nil, err
	}

	if req.ProcessingConfig == "" {
		req.ProcessingConfig = "default"
	}
	if err := pkg.saveValue(ctx, "processingConfiguration", req.ProcessingConfig); err != nil {
		return nil, err
	}

	if err := pkg.updateActiveAgent(ctx, "TODO"); err != nil {
		return nil, err
	}

	// TODO: copy of transfer needs to happen asynchronously because we can't
	// block the user request. This should however be done within workflow for
	// better tracking.
	//
	// Perhaps for now we can have a pool to cap the total amount of work that
	// we are willing to do at any given time, i.e. to be nice with Storage.
	go func() (err error) {
		logger := logger

		defer func() {
			if err != nil {
				logger.Info("Oopsie!", "err", err)
			} else {
				logger.Info("Done!")
			}
		}()

		// Create temporary directory.
		tmpDir, err := os.MkdirTemp(filepath.Join(sharedDir, "tmp"), "")
		if err != nil {
			return err
		}
		_ = os.Chmod(tmpDir, os.FileMode(0o770))
		logger = logger.WithValues("tmpDir", tmpDir)

		// Copy into new location.
		path, err := copyTransfer(ctx, ssclient, sharedDir, tmpDir, req.Name, req.Path[0])
		if err != nil {
			return fmt.Errorf("copy transfer: %v", err)
		}
		pkg.UpdatePath(path)
		logger = logger.WithValues("path", path)
		if err := store.UpdateTransferLocation(ctx, pkg.id, path); err != nil {
			logger.Info("Unable to update the transfer location.", "id", pkg.id, "path", path, "err", err)
		}

		logger.V(2).Info("Transfer ready for processing.")

		queue(pkg) // Start work.

		return nil
	}() // nolint:errcheck

	return pkg, nil
}

// ID returns the identifier of the package.
func (p *Package) ID() uuid.UUID {
	return p.id
}

// Path returns the real (no share dir vars) path to the package.
func (p *Package) Path() string {
	return strings.Replace(p.path, "%sharedPath%", p.sharedDir, 1)
}

func (p *Package) UpdatePath(path string) {
	p.path = strings.Replace(path, "%sharedPath%", p.sharedDir, 1)
}

// PathForDB returns the path to the package, as stored in the database.
func (p *Package) PathForDB() string {
	return strings.Replace(p.path, p.sharedDir, "%sharedPath%", 1)
}

// Name returns the package name derived from its dirname.
func (p *Package) Name() string {
	name := filepath.Base(filepath.Clean(p.Path()))
	return strings.Replace(name, "-"+p.id.String(), "", 1)
}

// String implements fmt.Stringer.
func (p *Package) String() string {
	return p.Name()
}

// parseProcessingConfig returns a list of preconfigured choices. A missing
// configuration file is a non-error, i.e. returns an empty slice of choices.
func (p *Package) parseProcessingConfig() ([]workflow.Choice, error) {
	f, err := os.Open(filepath.Join(p.path, "processingMCP.xml"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	choices, err := workflow.ParseConfig(f)
	if err != nil {
		return nil, fmt.Errorf("parse: %v", err)
	}

	return choices, nil
}

// PreconfiguredChoice looks up a pre-configured choice in the processing
// configuration file that is part of the package.
func (p *Package) PreconfiguredChoice(linkID uuid.UUID) (string, error) {
	// TODO: auto-approval should only happen if requested by the user, but
	// this is convenient during initial development.
	if chainID := Transfers.Decide(linkID); chainID != uuid.Nil {
		return chainID.String(), nil
	}

	choices, err := p.parseProcessingConfig()
	if err != nil {
		return "", err
	} else if len(choices) == 0 {
		return "", nil
	}

	var ret string
	for _, choice := range choices {
		if choice.LinkID() == linkID {
			ret = choice.GoToChain
			break
		}
	}

	return ret, nil
}

// Files iterates over all files associated with the package or that should be
// associated with a package, i.e. it first yields files based on database
// records verified to exist on the filesystem, then yields additional files
// found through filesystem traversal that meet specified filters.
//
// Parameters:
//   - filterFilenameEnd: the function filters files whose names end with
//     the specified suffix.
//   - filterSubdir: the function limits the search to files within
//     the specified subdirectory.
func (p *Package) Files(ctx context.Context, filterFilenameEnd, filterSubdir string) ([]replacementMapping, error) {
	files, err := p.store.Files(ctx, p.id, p.packageType(), filterFilenameEnd, filterSubdir, p.replacementPath())
	if err != nil {
		return nil, err
	}
	ret := make([]replacementMapping, 0, len(files))
	seen := make(map[string]struct{}, len(files))

	for _, f := range files {
		mapping := fileReplacements(p, &f)
		inputFile, ok := mapping["%inputFile%"]
		if !ok {
			continue
		}
		if _, err := os.Stat(string(inputFile)); errors.Is(err, os.ErrNotExist) {
			continue
		}
		seen[string(inputFile)] = struct{}{}
		ret = append(ret, mapping)
	}

	startPath := p.Path()
	if filterSubdir != "" {
		startPath += filterSubdir
	}
	err = filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fname := d.Name()
		if filterFilenameEnd != "" && !strings.HasPrefix(fname, filterFilenameEnd) {
			return nil
		}
		if _, ok := seen[path]; ok {
			return nil
		}
		ret = append(ret, map[string]replacement{
			"%relativeLocation": replacement(path),
			"%fileUUID%":        replacement("None"),
			"%fileGrpUse%":      replacement(""),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk dir: %v", err)
	}

	return ret, nil
}

func (p *Package) replacements() replacementMapping {
	return map[string]replacement{
		"%tmpDirectory%":        replacement(joinPath(p.sharedDir, "tmp", "")),
		"%processingDirectory%": replacement(joinPath(p.sharedDir, "currentlyProcessing", "")),
		"%watchDirectoryPath%":  replacement(joinPath(p.sharedDir, "watchedDirectories", "")),
		"%rejectedDirectory%":   replacement(joinPath(p.sharedDir, "rejected", "")),
	}
}

// saveValue persists "value" as a package variable.
func (p *Package) saveValue(ctx context.Context, name, value string) error {
	if err := p.store.CreateUnitVar(ctx, p.id, p.packageType(), name, value, uuid.Nil, true); err != nil {
		return fmt.Errorf("save value: %v", err)
	}
	return nil
}

// saveLinkID persist "linkID" as a package variable.
func (p *Package) saveLinkID(ctx context.Context, name string, linkID uuid.UUID) error {
	if err := p.store.CreateUnitVar(ctx, p.id, p.packageType(), name, "", linkID, true); err != nil {
		return fmt.Errorf("save linkID: %v", err)
	}
	return nil
}

func (p *Package) markAsProcessing(ctx context.Context) error {
	return p.store.UpdatePackageStatus(ctx, p.id, p.packageType(), enums.PackageStatusProcessing)
}

func (p *Package) markAsDone(ctx context.Context) error {
	return p.store.UpdatePackageStatus(ctx, p.id, p.packageType(), enums.PackageStatusDone)
}

func (p *Package) markAsFailed(ctx context.Context) error {
	return p.store.UpdatePackageStatus(ctx, p.id, p.packageType(), enums.PackageStatusFailed)
}

func (p *Package) updateActiveAgent(ctx context.Context, userID string) error {
	return nil // TODO: we have not implemented auth yet!
}

// unit represents logic that is specific to a particular type of package, e.g. Transfer.
type unit interface {
	// hydrate assits NewPackage in creating a package record in the database.
	hydrate(ctx context.Context, path, watchedDir string) error

	// reload populates some local state from the database records.
	reload(ctx context.Context) error

	// replacements returns a map of replacements for this package type.
	replacements(filterSubdirPath string) replacementMapping

	// replacementPath returns the replacement string for this package type.
	replacementPath() string

	// packageType describes the type of package.
	packageType() enums.PackageType

	// jobUnitType returns a string used to relate a Job to a package type in
	// the database. For example, "unitTransfer" is used to relate a Job to a
	// Transfer, whereas "unitDIP" relates the Job to a DIP.
	jobUnitType() string
}

type Transfer struct {
	pkg                     *Package
	processingConfiguration string
}

var _ unit = (*Transfer)(nil)

func (u *Transfer) hydrate(ctx context.Context, path, watchedDir string) error {
	path = joinPath(strings.Replace(path, u.pkg.sharedDir, "%sharedPath%", 1), "")
	id := uuidFromPath(path)
	created := false

	// Ensure that a Transfer is either created or updated. The strategy differs
	// depending on whether we know both its identifier and location, or only
	// the latter.
	if id != uuid.Nil {
		var opErr error
		created, opErr = u.pkg.store.UpsertTransfer(ctx, id, path)
		if opErr != nil {
			return opErr
		}
	} else {
		var opErr error
		id, created, opErr = u.pkg.store.EnsureTransfer(ctx, path)
		if opErr != nil {
			return opErr
		}
	}

	u.pkg.id = id
	u.pkg.path = path
	u.pkg.logger.V(1).Info("Transfer hydrated.", "created", created, "id", id)

	return nil
}

func (u *Transfer) reload(ctx context.Context) error {
	path, err := u.pkg.store.ReadTransferLocation(ctx, u.pkg.id)
	if err != nil {
		return err
	}
	u.pkg.UpdatePath(path)

	name, err := u.pkg.store.ReadUnitVar(ctx, u.pkg.id, u.packageType(), "processingConfiguration")
	if errors.Is(err, store.ErrNotFound) {
		u.processingConfiguration = "default"
	} else if err != nil {
		return err
	} else {
		u.processingConfiguration = name
	}

	return nil
}

func (u *Transfer) replacements(filterSubdirPath string) replacementMapping {
	mapping := u.pkg.replacements()
	maps.Copy(mapping, baseReplacements(u.pkg))
	maps.Copy(mapping, map[string]replacement{
		u.replacementPath():         replacement(u.pkg.Path()),
		"%unitType%":                replacement(u.packageType()),
		"%processingConfiguration%": replacement(u.processingConfiguration),
	})

	return mapping
}

func (u *Transfer) replacementPath() string {
	return "%transferDirectory%"
}

func (u *Transfer) packageType() enums.PackageType {
	return enums.PackageTypeTransfer
}

func (u *Transfer) jobUnitType() string {
	return "unitTransfer"
}

type SIP struct {
	pkg         *Package
	sipType     string
	aipFilename string
}

var _ unit = (*SIP)(nil)

func (u *SIP) hydrate(ctx context.Context, path, watchedDir string) error {
	path = joinPath(strings.Replace(path, u.pkg.sharedDir, "%sharedPath%", 1), "")
	id := uuidFromPath(path)
	created := false

	// Ensure that a SIP is either created or updated. The strategy differs
	// depending on whether we know both its identifier and location, or only
	// the latter.
	if id != uuid.Nil {
		var opErr error
		created, opErr = u.pkg.store.UpsertSIP(ctx, id, path)
		if opErr != nil {
			return opErr
		}
	} else {
		var opErr error
		id, created, opErr = u.pkg.store.EnsureSIP(ctx, path)
		if opErr != nil {
			return opErr
		}
	}

	// SIP package is a partial (objects or metadata-only) reingest.
	// Full reingests use a different workflow chain.
	if strings.Contains(watchedDir, "system/reingestAIP") {
		if err := u.pkg.saveValue(ctx, "isPartialReingest", "true"); err != nil {
			return err
		}
	}

	u.pkg.id = id
	u.pkg.path = path
	u.pkg.logger.V(1).Info("SIP hydrated.", "created", created, "id", id)

	return nil
}

func (u *SIP) reload(ctx context.Context) error {
	sip, err := u.pkg.store.ReadSIP(ctx, u.pkg.id)
	if err != nil {
		return err
	}

	u.pkg.UpdatePath(sip.CurrentPath)
	u.aipFilename = sip.AIPFilename
	u.sipType = sip.Type

	return nil
}

func (u *SIP) replacements(filterSubdirPath string) replacementMapping {
	mapping := u.pkg.replacements()
	maps.Copy(mapping, baseReplacements(u.pkg))
	maps.Copy(mapping, map[string]replacement{
		"%unitType%":   replacement(u.packageType()),
		"%AIPFilename": replacement(u.aipFilename),
		"%SIPType%":    replacement(u.sipType),
	})
	return mapping
}

func (u *SIP) replacementPath() string {
	return "%SIPDirectory%"
}

func (u *SIP) packageType() enums.PackageType {
	return enums.PackageTypeSIP
}

func (u *SIP) jobUnitType() string {
	return "unitSIP"
}

type DIP struct {
	pkg *Package
}

var _ unit = (*DIP)(nil)

func (u *DIP) hydrate(ctx context.Context, path, watchedDir string) error {
	path = joinPath(strings.Replace(path, u.pkg.sharedDir, "%sharedPath%", 1), "")
	id := uuidFromPath(path)
	created := false

	if id != uuid.Nil {
		var opErr error
		created, opErr = u.pkg.store.UpsertDIP(ctx, id, path)
		if opErr != nil {
			return opErr
		}
	} else {
		var opErr error
		id, created, opErr = u.pkg.store.EnsureDIP(ctx, path)
		if opErr != nil {
			return opErr
		}
	}

	u.pkg.id = id
	u.pkg.path = path
	u.pkg.logger.V(1).Info("DIP hydrated.", "created", created, "id", id)

	return nil
}

func (u *DIP) reload(ctx context.Context) error {
	return nil // No-op.
}

func (u *DIP) replacements(filterSubdirPath string) replacementMapping {
	mapping := u.pkg.replacements()
	maps.Copy(mapping, baseReplacements(u.pkg))
	maps.Copy(mapping, map[string]replacement{
		"%unitType%": replacement(u.packageType()),
	})
	if filterSubdirPath != "" {
		mapping["%relativeLocation%"] = replacement(
			strings.Replace(filterSubdirPath, "%sharedPath%", u.pkg.sharedDir, 1),
		)
	}

	return mapping
}

func (u *DIP) replacementPath() string {
	return "%SIPDirectory%"
}

func (u *DIP) packageType() enums.PackageType {
	return enums.PackageTypeDIP
}

func (u *DIP) jobUnitType() string {
	return "unitDIP"
}

func dirBasename(path string) string {
	abs, _ := filepath.Abs(path)
	return filepath.Base(abs)
}

// baseReplacements returns replacements needed by all unit types.
func baseReplacements(p *Package) replacementMapping {
	path := p.Path()

	return map[string]replacement{
		"%SIPUUID%":              replacement(p.id.String()),
		"%SIPName%":              replacement(p.Name()),
		"%SIPLogsDirectory%":     replacement(joinPath(path, "logs", "")),
		"%SIPObjectsDirectory%":  replacement(joinPath(path, "objects", "")),
		"%SIPDirectory%":         replacement(path),
		"%SIPDirectoryBasename%": replacement(dirBasename(path)),
		"%relativeLocation%":     replacement(p.PathForDB()),
	}
}

func fileReplacements(pkg *Package, f *store.File) replacementMapping {
	mapping := map[string]replacement{}
	maps.Copy(mapping, baseReplacements(pkg))

	dirName := filepath.Dir(f.CurrentLocation)
	ext := filepath.Ext(f.CurrentLocation)
	extWithDot := "." + ext
	name := filepath.Base(strings.TrimSuffix(f.CurrentLocation, ext))
	absolutePath := strings.ReplaceAll(f.CurrentLocation, "%SIPDirectory%", joinPath(pkg.Path(), ""))
	absolutePath = strings.ReplaceAll(absolutePath, "%transferDirectory%", joinPath(pkg.Path(), ""))

	maps.Copy(mapping, map[string]replacement{
		"%fileUUID%":             replacement(f.ID.String()),
		"%originalLocation%":     replacement(f.OriginalLocation),
		"%currentLocation%":      replacement(f.CurrentLocation),
		"%fileGrpUse%":           replacement(f.FileGrpUse),
		"%fileDirectory%":        replacement(dirName),
		"%fileName%":             replacement(name),
		"%fileExtension%":        replacement(ext),
		"%fileExtensionWithDot%": replacement(extWithDot),

		// TODO: standardize duplicates (the last two are used in the FPR db).
		"%relativeLocation%": replacement(absolutePath),
		"%inputFile%":        replacement(absolutePath),
		"%fileFullName%":     replacement(absolutePath),
	})

	return mapping
}

type replacement string

// escape special characters like slashes, quotes, and backticks.
func (r replacement) escape() string {
	v := string(r)

	// Escape backslashes first
	v = strings.ReplaceAll(v, "\\", "\\\\")

	var escaped string
	for _, char := range v {
		switch char {
		case '\\':
			escaped += "\\\\"
		case '"', '`':
			escaped += "\\" + string(char)
		default:
			escaped += string(char)
		}
	}

	return escaped
}

type replacementMapping map[string]replacement

// copy returns a new map with copied replacements.
func (rm replacementMapping) copy() replacementMapping {
	n := map[string]replacement{}
	for k, v := range rm {
		n[k] = v
	}

	return n
}

// update the replacements with the context values of a given chain.
func (rm replacementMapping) update(c *chain) replacementMapping {
	for el := c.context.Front(); el != nil; el = el.Next() {
		rm[el.Key] = replacement(el.Value)
	}

	return rm
}

// with returns a copy of the mapping with input merged into it.
func (rm replacementMapping) with(input replacementMapping) replacementMapping {
	ret := rm.copy()
	for k, v := range input {
		ret[k] = v
	}

	return ret
}

func (rm replacementMapping) replaceValues(input string) string {
	const defaultSharedDir = "/var/archivematica/sharedDirectory"
	if input == "" {
		return ""
	}

	for k, v := range rm {
		escaped := v.escape()

		// Unfortunately MCPClient expects paths to sit under "/var/archivematica/sharedDirectory".
		// If CCP is using a different basepath, we have to replace it otherwise MCPClient won't find it.
		sep := string(filepath.Separator)
		if strings.HasPrefix(escaped, sep) {
			tailed := strings.HasSuffix(escaped, sep)
			parts := strings.Split(escaped, sep)
			if len(parts) > 3 && parts[1] == "tmp" && strings.HasPrefix(parts[2], "ccp-sharedDir") {
				escaped = joinPath(defaultSharedDir, filepath.Join(parts[3:]...))
				if tailed {
					escaped += sep
				}
			}
		}

		input = strings.ReplaceAll(input, k, escaped)
	}

	return input
}
