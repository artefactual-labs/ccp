package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

const maxConcurrentPackages = 2

type Controller struct {
	logger logr.Logger

	store store.Store

	// Embedded job server compatible with Gearman.
	gearman *gearmin.Server

	// wf is the workflow document.
	wf *workflow.Document

	sharedDir string

	watchedDir string

	// activePackages is the list of active packages.
	activePackages []*Package

	// queuedPackages is the list of queued packages, FIFO style.
	queuedPackages []*Package

	// sync.Mutex protects the internal Package slices.
	mu sync.Mutex

	// group is a collection of goroutines used for processing packages.
	group *errgroup.Group

	// groupCtx is the context associated to the errgroup.
	groupCtx context.Context

	// groupCancel tells active goroutines in the errgroup to abandon.
	groupCancel context.CancelFunc

	// closeOnce guarantees that the closing procedure runs only once.
	closeOnce sync.Once
}

func New(logger logr.Logger, store store.Store, gearman *gearmin.Server, wf *workflow.Document, sharedDir, watchedDir string) *Controller {
	c := &Controller{
		logger:         logger,
		store:          store,
		gearman:        gearman,
		wf:             wf,
		sharedDir:      sharedDir,
		watchedDir:     watchedDir,
		activePackages: []*Package{},
		queuedPackages: []*Package{},
	}

	c.groupCtx, c.groupCancel = context.WithCancel(context.Background())
	c.group, _ = errgroup.WithContext(c.groupCtx)
	c.group.SetLimit(10)

	return c
}

// Run tries to start processing queued transfers.
func (c *Controller) Run() error {
	go func() {
		ticker := time.NewTicker(time.Second / 4)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.pick()
			case <-c.groupCtx.Done():
				return
			}
		}
	}()

	return nil
}

// Submit a new package to the queue.
func (c *Controller) Submit(ctx context.Context, req *adminv1.CreatePackageRequest) (*Package, error) {
	// 1. Create Package (Transfer).
	//    transfer = models.Transfer.objects.create(**kwargs)
	//    if not processing_configuration_file_exists(processing_config): processing_config = "default"
	//    transfer.set_processing_configuration(processing_config)
	//    transfer.update_active_agent(user_id)
	// 2. Create temporary directory inside sharedDir/tmp.
	//    tmpdir = mkdtemp(dir=os.path.join(_get_setting("SHARED_DIRECTORY"), "tmp"))
	// 3. Identify starting point.
	//    starting_point = PACKAGE_TYPE_STARTING_POINTS.get(type_)
	// 4. Start creation.
	//    params = (transfer, name, path, tmpdir, starting_point)
	//    if auto_approve:
	//        params = params + (workflow, package_queue)
	//        result = executor.submit(_start_package_transfer_with_auto_approval, *params)
	//    else:
	//        result = executor.submit(_start_package_transfer, *params)
	// 5. Adjust permissions?
	//    result.add_done_callback(lambda f: os.chmod(tmpdir, 0o770))

	pkg, err := NewTransferPackage(c.groupCtx, c.logger.WithName("package"), c.store, c.sharedDir, req)
	if err != nil {
		return nil, fmt.Errorf("create package: %v", err)
	}

	c.queue(pkg)
	c.pick() // Start work right away, we don't want to wait for the next tick.

	return pkg, nil
}

// Notify the controller of a new with a slice of filesystem events.
func (c *Controller) Notify(path string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("notify: %v", err)
		}
	}()

	rel, err := filepath.Rel(c.watchedDir, path)
	if err != nil {
		return err
	}

	dir, _ := filepath.Split(rel)
	dir = trim(dir)

	var wd *workflow.WatchedDirectory
	for _, item := range c.wf.WatchedDirectories {
		if trim(item.Path) == dir {
			wd = item
			break
		}
	}
	if wd == nil {
		return fmt.Errorf("unmatched event")
	}

	c.logger.V(2).Info("Identified new package.", "path", path, "type", wd.UnitType)

	logger := c.logger.WithName("package").WithValues("wd", wd.Path, "path", path)
	if pkg, err := NewPackage(c.groupCtx, logger, c.store, c.sharedDir, path, wd); err != nil {
		return err
	} else {
		c.queue(pkg)
		c.pick() // Start work right away, we don't want to wait for the next tick.
	}

	return nil
}

func (c *Controller) queue(pkg *Package) {
	c.mu.Lock()
	c.queuedPackages = append(c.queuedPackages, pkg)
	c.mu.Unlock()
}

func (c *Controller) pick() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.activePackages) == maxConcurrentPackages {
		c.logger.V(2).Info("Not accepting new packages at this time.", "active", len(c.activePackages), "max", maxConcurrentPackages)
		return
	}

	var current *Package
	if len(c.queuedPackages) > 0 {
		current = c.queuedPackages[0]
		c.activePackages = append(c.activePackages, current)
		c.queuedPackages = c.queuedPackages[1:]
	}

	if current == nil {
		return
	}

	c.group.Go(func() error {
		logger := c.logger.V(2).WithValues("package", current)

		defer c.deactivate(current)

		logger.Info("Processing started.")
		err := NewIterator(logger, c.gearman, c.wf, current).Process(c.groupCtx) // Block.
		if err != nil {
			logger.Info("Processing failed.", "err", err)
		} else {
			logger.Info("Processing completed successfully")
		}

		return err
	})
}

// deactivate removes a package from the activePackages queue.
func (c *Controller) deactivate(p *Package) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, item := range c.activePackages {
		if item.id == p.id {
			c.activePackages = append(c.activePackages[:i], c.activePackages[i+1:]...)
			break
		}
	}
}

// Active lists all active packages.
func (c *Controller) Active() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	ret := make([]string, len(c.activePackages))
	for i, item := range c.activePackages {
		ret[i] = item.String()
	}

	return ret
}

// Decisions lists awaiting decisions for all active packages.
func (c *Controller) Decisions() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	ret := []string{}

	for _, item := range c.activePackages {
		opts := item.Decision()
		ln := len(opts)
		if ln == 0 {
			continue
		}
		ret = append(ret, fmt.Sprintf("package %s has an awaiting decision with %d options available", item, ln))

	}

	return ret
}

func (c *Controller) Close() error {
	var err error

	c.closeOnce.Do(func() {
		c.groupCancel()
		err = c.group.Wait()
	})

	return err
}

func trim(path string) string {
	return strings.Trim(path, string(filepath.Separator))
}
