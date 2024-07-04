package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/authn"
	"github.com/artefactual-labs/gearmin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual/archivematica/hack/ccp/internal/derrors"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

const maxConcurrentPackages = 2

// Controller manages concurrent processing of packages.
//
// There are three queues: queued, active and awaiting.
type Controller struct {
	logger logr.Logger

	// Application metrics.
	metrics *metrics.Metrics

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

	// awaitingPackages is the list packages awaiting a decision indexed by the
	// package identifier.
	awaitingPackages map[uuid.UUID][]*decision

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

func New(logger logr.Logger, metrics *metrics.Metrics, ssclient ssclient.Client, store store.Store, gearman *gearmin.Server, wf *workflow.Document, sharedDir, watchedDir string) *Controller {
	c := &Controller{
		logger:           logger,
		metrics:          metrics,
		ssclient:         ssclient,
		store:            store,
		gearman:          gearman,
		wf:               wf,
		sharedDir:        sharedDir,
		watchedDir:       watchedDir,
		activePackages:   []*Package{},
		queuedPackages:   []*Package{},
		awaitingPackages: map[uuid.UUID][]*decision{},
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

	pkg, err := NewTransferPackage( //nolint: contextcheck
		// ctx is request-scoped, use the group context instead.
		authn.SetInfo(c.groupCtx, authn.GetInfo(ctx)),
		c.logger.WithName("package"),
		c.store,
		c.ssclient,
		c.sharedDir,
		req,
		queue,
	)
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
	c.metrics.PackageQueueLengthGauge.WithLabelValues(pkg.packageType().String()).Inc()
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
		c.metrics.ActivePackageGauge.Inc()
	}

	if pkg == nil {
		return
	}

	c.group.Go(func() error {
		logger := c.logger.V(2).WithValues("package", pkg)
		logger.Info("Processing started.")
		defer c.deactivate(pkg)

		iter := newJobIterator(c.groupCtx, logger, c.metrics, c.gearman, c.wf, pkg)
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
func (c *Controller) deactivate(pkg *Package) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, item := range c.activePackages {
		if item.id == pkg.id {
			c.activePackages = append(c.activePackages[:i], c.activePackages[i+1:]...)
			c.metrics.ActivePackageGauge.Dec()
			c.metrics.PackageQueueLengthGauge.WithLabelValues(pkg.packageType().String()).Dec()
			break
		}
	}
}

// await blocks until the awaiting package is resolved.
func (c *Controller) await(iter *jobIterator, pkg *Package, dec *decision) error {
	_ = c.queueToAwait(pkg, dec)
	defer c.dequeueFromAwait(pkg, dec)

	next, err := dec.await(c.groupCtx)
	c.logger.Info("Resolution of awaiting package completed.", "next", next, "err", err)
	if err != nil {
		return err
	}

	iter.nextLink = next

	return nil
}

// queueToAwait moves an active package to the awaiting list.
func (c *Controller) queueToAwait(pkg *Package, dec *decision) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Confirm that the package is in the active queue.
	var pos *int
	for i, active := range c.activePackages {
		if active.id == pkg.id {
			pos = &i
			break
		}
	}
	if pos == nil {
		return errors.New("package not found in the active list")
	}

	// Remove from the active list.
	c.activePackages = append(c.activePackages[:*pos], c.activePackages[*pos+1:]...)

	// Add to the awaiting list.
	if l, ok := c.awaitingPackages[pkg.id]; !ok {
		c.awaitingPackages[pkg.id] = []*decision{dec}
	} else {
		c.awaitingPackages[pkg.id] = append(l, dec)
	}

	return nil
}

func (c *Controller) dequeueFromAwait(pkg *Package, dec *decision) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Retrieve the list of decisions of this package.
	decisions, ok := c.awaitingPackages[pkg.id]
	if !ok {
		return
	}

	// Find the position of the decision that we want to remove.
	var pos *int
	for i, item := range decisions {
		if item.id == dec.id {
			pos = &i
			break
		}
	}

	if pos == nil {
		return
	}

	// Remove the decision from the slice.
	decisions = append(decisions[:*pos], decisions[*pos+1:]...)

	// Update the awaitingPackages map.
	if len(decisions) == 0 {
		delete(c.awaitingPackages, pkg.id)
	} else {
		c.awaitingPackages[pkg.id] = decisions
	}

	// Add package back to the active list.
	c.activePackages = append(c.activePackages, pkg)
}

func (c *Controller) Active(id uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, item := range c.activePackages {
		if item.id == id {
			return true
		}
	}

	return false
}

// ActivePackages lists all active packages.
func (c *Controller) ActivePackages() []uuid.UUID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ret := make([]uuid.UUID, 0, len(c.activePackages))
	for _, pkg := range c.activePackages {
		ret = append(ret, pkg.ID())
	}

	return ret
}

// PackageDecisions returns the awaiting decisions of a given package.
func (c *Controller) PackageDecisions(pkgID uuid.UUID) ([]*adminv1.Decision, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	decisions, ok := c.awaitingPackages[pkgID]
	if !ok {
		return nil, false
	}

	ret := make([]*adminv1.Decision, len(decisions))
	for i, dec := range decisions {
		ret[i] = dec.convert()
	}

	return ret, true
}

// Decisions lists awaiting decisions for all active packages.
func (c *Controller) Decisions() map[uuid.UUID][]*adminv1.Decision {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ret := make(map[uuid.UUID][]*adminv1.Decision, len(c.awaitingPackages))

	for pkgID, decList := range c.awaitingPackages {
		decisions := make([]*adminv1.Decision, len(decList))
		for i, dec := range decList {
			decisions[i] = dec.convert()
		}
		ret[pkgID] = decisions
	}

	return ret
}

func (c *Controller) ResolveDecision(ctx context.Context, id uuid.UUID, pos int) (err error) {
	defer derrors.Add(&err, "ResolveDecision()")

	c.mu.RLock()
	defer c.mu.RUnlock()

	var match *decision
	for _, decisions := range c.awaitingPackages {
		for _, dec := range decisions {
			if dec.id == id {
				match = dec
				break
			}
		}
	}

	if match == nil {
		return errors.New("decision cannot be found")
	}

	if err := match.pkg.updateActiveAgent(ctx); err != nil {
		return fmt.Errorf("update active agent: %v", err)
	}

	return match.resolveWithPos(pos)
}

func (c *Controller) ResolveDecisionLegacy(ctx context.Context, jobID uuid.UUID, choice string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var match *decision
	for _, decisions := range c.awaitingPackages {
		for _, dec := range decisions {
			if dec.jobID == jobID {
				match = dec
				break
			}
		}
	}

	if match == nil {
		return errors.New("package is not awaiting")
	}

	if err := match.pkg.updateActiveAgent(ctx); err != nil {
		return fmt.Errorf("update active agent: %v", err)
	}

	// We attempt to read the choice as an integer describing the position of
	// the decision to choose.
	if pos, err := strconv.Atoi(choice); err == nil {
		return match.resolveWithPos(pos)
	}

	return match.resolveWithChoice(choice)
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
