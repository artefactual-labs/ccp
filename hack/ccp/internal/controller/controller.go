package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

const maxConcurrentPackages = 2

type Controller struct {
	logger logr.Logger

	// Archivematica Storage Service API client.
	ssclient ssclient.Client

	// Application store.
	store store.Store

	// Embedded job server compatible with Gearman.
	gearman *gearmin.Server

	// wf is the workflow document.
	wf *workflow.Document

	// Archivematica shared directory.
	sharedDir string

	// Archivematica watched directory.
	watchedDir string

	// activePackages is the list of active packages.
	activePackages []*Package

	// queuedPackages is the list of queued packages, FIFO style.
	queuedPackages []*Package

	// awaitingPackages is the list of packages awaiting a decision.
	awaitingPackages map[uuid.UUID]*decision

	// sync.RWMutex protects the internal Package slices.
	mu sync.RWMutex

	// group is a collection of goroutines used for processing packages.
	group *errgroup.Group

	// groupCtx is the context associated to the errgroup.
	groupCtx context.Context

	// groupCancel tells active goroutines in the errgroup to abandon.
	groupCancel context.CancelFunc

	// closeOnce guarantees that the closing procedure runs only once.
	closeOnce sync.Once
}

func New(logger logr.Logger, ssclient ssclient.Client, store store.Store, gearman *gearmin.Server, wf *workflow.Document, sharedDir, watchedDir string) *Controller {
	c := &Controller{
		logger:           logger,
		ssclient:         ssclient,
		store:            store,
		gearman:          gearman,
		wf:               wf,
		sharedDir:        sharedDir,
		watchedDir:       watchedDir,
		activePackages:   []*Package{},
		queuedPackages:   []*Package{},
		awaitingPackages: map[uuid.UUID]*decision{},
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

// Submit a transfer request.
func (c *Controller) Submit(ctx context.Context, req *adminv1.CreatePackageRequest) (*Package, error) {
	// TODO: have NewTransferPackage return a function we can schedule here.
	var once sync.Once
	queue := func(pkg *Package) {
		once.Do(func() {
			c.queue(pkg)
			c.pick() // Start work right away, we don't want to wait for the next tick.
		})
	}

	pkg, err := NewTransferPackage(c.groupCtx, c.logger.WithName("package"), c.store, c.ssclient, c.sharedDir, req, queue)
	if err != nil {
		return nil, fmt.Errorf("create package: %v", err)
	}

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

	var pkg *Package
	if len(c.queuedPackages) > 0 {
		pkg = c.queuedPackages[0]
		c.activePackages = append(c.activePackages, pkg)
		c.queuedPackages = c.queuedPackages[1:]
	}

	if pkg == nil {
		return
	}

	c.group.Go(func() error {
		logger := c.logger.V(2).WithValues("package", pkg)
		logger.Info("Processing started.")
		defer c.deactivate(pkg)

		iter := newJobIterator(c.groupCtx, logger, c.gearman, c.wf, pkg)
		for {
			err := iter.next() // Runs the next job.

			if errors.Is(err, errEnd) || errors.Is(err, io.EOF) {
				return nil
			} else if ew, ok := isErrWait(err); ok {
				if err := c.await(iter, pkg, ew.decision); err != nil {
					return err
				} else {
					continue
				}
			} else if err != nil {
				logger.Error(err, "Processing failed.")
				return err
			}
		}
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

// await blocks until the awaiting package is resolved.
func (c *Controller) await(iter *jobIterator, pkg *Package, decision *decision) error {
	_ = c.queueToAwait(pkg, decision)
	defer c.dequeueFromAwait(pkg)

	next, err := decision.await(c.groupCtx)
	c.logger.Info("Resolution of awaiting package completed.", "next", next, "err", err)
	if err != nil {
		return err
	}

	iter.nextLink = next

	return nil
}

// queueToAwait moves an active package to the awaiting list.
func (c *Controller) queueToAwait(pkg *Package, decision *decision) error {
	pkgID := pkg.id

	// Confirm that the package is in the active queue.
	c.mu.RLock()
	var index int
	var found bool
	for i, active := range c.activePackages {
		if active.id == pkgID {
			index = i
			found = true
			break
		}
	}
	c.mu.RUnlock()
	if !found {
		return errors.New("package not found in the active list")
	}

	// Move to the awaiting list.
	c.mu.Lock()
	c.activePackages = append(c.activePackages[:index], c.activePackages[index+1:]...)
	c.awaitingPackages[pkgID] = decision
	c.mu.Unlock()

	return nil
}

func (c *Controller) dequeueFromAwait(pkg *Package) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.awaitingPackages[pkg.id]
	if !ok {
		return
	}

	delete(c.awaitingPackages, pkg.id)
	c.activePackages = append(c.activePackages, pkg)
}

func (c *Controller) IsPackageActive(id uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, item := range c.activePackages {
		if item.id == id {
			return true
		}
	}

	return false
}

// Decision returns the decision of a Package given its identifier.
func (c *Controller) Decision(id uuid.UUID) (*adminv1.Decision, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	decision, ok := c.awaitingPackages[id]
	if !ok {
		return nil, false
	}

	ret := &adminv1.Decision{
		Name:   decision.name,
		Choice: make([]*adminv1.Choice, 0, len(decision.choices)),
	}
	for i, item := range decision.choices {
		choice := &adminv1.Choice{
			Id:    int32(i),
			Label: item.label,
		}
		ret.Choice = append(ret.Choice, choice)
	}

	return ret, true
}

// Active lists all active packages.
func (c *Controller) Active() []uuid.UUID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ret := make([]uuid.UUID, 0, len(c.activePackages))
	for _, pkg := range c.activePackages {
		ret = append(ret, pkg.ID())
	}

	return ret
}

// Decisions lists awaiting decisions for all active packages.
func (c *Controller) Decisions() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ret := make([]string, 0, len(c.activePackages))
	for _, decision := range c.awaitingPackages {
		ret = append(ret, fmt.Sprintf("%s: %s", decision.pkg.id, decision.name))
	}

	return ret
}

func (c *Controller) ResolveDecision(pkgID uuid.UUID, pos int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var match *decision
	for id, decision := range c.awaitingPackages {
		if id == pkgID {
			match = decision
			break
		}
	}

	if match == nil {
		return errors.New("package is not awaiting")
	}

	return match.resolve(pos)
}

func (c *Controller) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.groupCancel()
		if waitErr := c.group.Wait(); errors.Is(waitErr, context.Canceled) {
			err = waitErr
		}
	})

	return err
}
