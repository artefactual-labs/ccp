package ssclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	ssclientlib "go.artefactual.dev/ssclient"
	"go.artefactual.dev/ssclient/kiota"

	"github.com/artefactual/archivematica/hack/ccp/internal/store"
)

type Pipeline struct {
	ID  uuid.UUID
	URI string
}

type Location struct {
	ID      uuid.UUID
	Purpose string
	Path    string
}

// Client wraps go.artefactual.dev/ssclient-go.
type Client interface {
	ReadPipeline(ctx context.Context, id uuid.UUID) (*Pipeline, error)
	ReadLocation(ctx context.Context, purpose string) ([]*Location, error)
	ReadDefaultLocation(ctx context.Context, purpose string) (*Location, error)
	CopyFiles(ctx context.Context, l *Location, files []string) error
}

// clientImpl implements Client. It uses the store to read the pipeline ID and
// it caches the pipeline details to avoid hitting the server too often.
type clientImpl struct {
	client *kiota.Client
	store  store.Store

	// Cached pipeline with the last retrieval timestamp and protected.
	p  *Pipeline
	ts time.Time
	mu sync.RWMutex
}

var _ Client = (*clientImpl)(nil)

func NewClient(store store.Store, baseURL, username, key string) (*clientImpl, error) {
	stdClient := retryablehttp.NewClient().StandardClient()
	k, err := ssclientlib.New(stdClient, baseURL, username, key)
	if err != nil {
		return nil, err
	}

	c := &clientImpl{client: k, store: store}

	return c, nil
}

func (c *clientImpl) ReadPipeline(ctx context.Context, id uuid.UUID) (*Pipeline, error) {
	m, err := c.client.Api().V2().Pipeline().ByUuid(id.String()).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	p := &Pipeline{
		URI: *m.GetResourceUri(),
	}

	if id, err := uuid.Parse(*m.GetUuid()); err != nil {
		return nil, err
	} else {
		p.ID = id
	}

	return p, nil
}

func (c *clientImpl) ReadLocation(ctx context.Context, purpose string) ([]*Location, error) {
	p, err := c.pipeline(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Println(p.ID)

	return nil, nil
}

func (c *clientImpl) ReadDefaultLocation(ctx context.Context, purpose string) (*Location, error) {
	return nil, nil
}

func (c *clientImpl) CopyFiles(ctx context.Context, l *Location, files []string) error {
	p, err := c.pipeline(ctx)
	if err != nil {
		return err
	}
	fmt.Println(p.URI)

	return nil
}

// pipeline returns the details of the current pipeline.
func (c *clientImpl) pipeline(ctx context.Context) (Pipeline, error) {
	const ttl = time.Second * 15

	c.mu.Lock()
	if c.p != nil && time.Since(c.ts) <= ttl {
		defer c.mu.Unlock()
		return *c.p, nil
	}
	c.mu.Unlock()

	pipelineID, err := c.store.ReadPipelineID(ctx)
	if err != nil {
		return Pipeline{}, err
	}
	p, err := c.ReadPipeline(ctx, pipelineID)
	if err != nil {
		return Pipeline{}, err
	}

	c.mu.RLock()
	c.p = p
	c.ts = time.Now()
	c.mu.Unlock()

	return *c.p, nil
}
