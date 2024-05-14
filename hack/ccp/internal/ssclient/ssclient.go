package ssclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	kiotaabs "github.com/microsoft/kiota-abstractions-go"
	kiotahttp "github.com/microsoft/kiota-http-go"
	ssclientlib "go.artefactual.dev/ssclient"
	"go.artefactual.dev/ssclient/kiota"
	"go.artefactual.dev/ssclient/kiota/api"
	"go.artefactual.dev/ssclient/kiota/models"
	"go.artefactual.dev/tools/ref"

	"github.com/artefactual/archivematica/hack/ccp/internal/derrors"
	"github.com/artefactual/archivematica/hack/ccp/internal/store"
)

var ErrLocationNotAvailable = errors.New("location not available")

type Pipeline struct {
	ID  uuid.UUID
	URI string
}

type Location struct {
	ID           uuid.UUID
	URI          string
	Purpose      string
	Path         string
	RelativePath string
	Pipelines    []string
}

// Client wraps go.artefactual.dev/ssclient-go. It provides additional
// functionality like awareness of the current pipeline identifier, the ability
// to page results and populate the default location.
type Client interface {
	ReadPipeline(ctx context.Context, id uuid.UUID) (*Pipeline, error)
	ReadDefaultLocation(ctx context.Context, purpose string) (*Location, error)
	ReadProcessingLocation(ctx context.Context) (*Location, error)
	ListLocations(ctx context.Context, path, purpose string) ([]*Location, error)

	// MoveFiles moves files between locations. `files` is a list of pairs
	// indicating the paths of the source file and its destination (both paths
	// must be relative to their Location of the files to be moved).
	MoveFiles(ctx context.Context, src, dst *Location, files [][2]string) error
}

// clientImpl implements Client.
type clientImpl struct {
	client *kiota.Client
	store  store.Store

	// Cached pipeline with the last retrieval timestamp and protected.
	p  *Pipeline
	ts time.Time
	mu sync.RWMutex
}

var _ Client = (*clientImpl)(nil)

func NewClient(httpClient *http.Client, store store.Store, config Config) (*clientImpl, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	k, err := ssclientlib.New(httpClient, config.BaseURL, config.Username, config.Key)
	if err != nil {
		return nil, err
	}

	c := &clientImpl{client: k, store: store}

	return c, nil
}

func (c *clientImpl) ReadPipeline(ctx context.Context, id uuid.UUID) (_ *Pipeline, err error) {
	derrors.Add(&err, "ReadPipeline(%s)", id)

	m, err := c.client.Api().V2().Pipeline().ByUuid(id.String()).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	p, err := convertPipeline(m)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *clientImpl) ReadDefaultLocation(ctx context.Context, purpose string) (_ *Location, err error) {
	derrors.Add(&err, "ReadDefaultLocation(%s)", purpose)

	p, err := c.pipeline(ctx)
	if err != nil {
		return nil, err
	}

	headerOptions := kiotahttp.NewHeadersInspectionOptions()
	headerOptions.InspectResponseHeaders = true

	reqConfig := &api.V2LocationDefaultWithPurposeItemRequestBuilderGetRequestConfiguration{
		Options: []kiotaabs.RequestOption{headerOptions},
	}
	if err := c.client.Api().V2().Location().DefaultEscaped().ByPurpose(purpose).Get(ctx, reqConfig); err != nil {
		return nil, err
	}

	uris := headerOptions.ResponseHeaders.Get("Location")
	if len(uris) < 1 {
		return nil, ErrLocationNotAvailable
	}
	uri := uris[0]
	if uri == "" {
		return nil, ErrLocationNotAvailable
	}

	// Capture the UUID in the URI, e.g. "/api/v2/location/be68cfa8-d32a-44ba-a140-2ec5d6b903e0/".
	id := strings.TrimSuffix(strings.TrimPrefix(uri, "/api/v2/location/"), "/")

	res, err := c.client.Api().V2().Location().ByUuid(id).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Confirm that the default location has been made available to this pipeline.
	var match bool
	for _, item := range res.GetPipeline() {
		if item == p.URI {
			match = true
			break
		}
	}
	if !match {
		return nil, ErrLocationNotAvailable
	}

	ret, err := convertLocation(res)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *clientImpl) ReadProcessingLocation(ctx context.Context) (_ *Location, err error) {
	derrors.Add(&err, "ReadProcessingLocation")

	res, err := c.ListLocations(ctx, "", models.CP_LOCATIONPURPOSE.String())
	if err != nil {
		return nil, err
	}

	if len(res) < 1 {
		return nil, ErrLocationNotAvailable
	}

	// We can have many but we'll return the first match.
	return res[0], nil
}

func (c *clientImpl) ListLocations(ctx context.Context, path, purpose string) (_ []*Location, err error) {
	derrors.Add(&err, "ListLocations(%s, %s)", path, purpose)

	p, err := c.pipeline(ctx)
	if err != nil {
		return nil, err
	}

	reqConfig := &api.V2LocationRequestBuilderGetRequestConfiguration{
		QueryParameters: &api.V2LocationRequestBuilderGetQueryParameters{
			Pipeline__uuid: ref.New(p.ID.String()),
			Limit:          ref.New(int32(100)),
		},
	}

	if path != "" {
		reqConfig.QueryParameters.Relative_path = &path
	}

	if purpose != "" {
		ps, err := models.ParseLocationPurpose(purpose)
		if err != nil {
			return nil, err
		}
		if mps, ok := ps.(*models.LocationPurpose); ok {
			reqConfig.QueryParameters.PurposeAsLocationPurpose = mps
		} else {
			return nil, fmt.Errorf("invalid purpose value: %v", ps)
		}
	}

	list, err := c.client.Api().V2().Location().Get(ctx, reqConfig)
	if err != nil {
		return nil, err
	}

	objects := list.GetObjects()
	ret := make([]*Location, 0, len(objects))
	for _, obj := range objects {
		l, err := convertLocation(obj)
		if err != nil {
			return nil, err
		}
		ret = append(ret, l)
	}

	return ret, nil
}

func (c *clientImpl) MoveFiles(ctx context.Context, src, dst *Location, files [][2]string) (err error) {
	derrors.Add(&err, "MoveFiles()")

	p, err := c.pipeline(ctx)
	if err != nil {
		return err
	}

	body := models.NewMoveRequest()
	body.SetPipeline(&p.URI)
	body.SetOriginLocation(&src.URI)

	moves := make([]models.MoveFileable, 0, len(files))
	for _, f := range files {
		m := models.NewMoveFile()
		m.SetSource(&f[0])
		m.SetDestination(&f[1])
		moves = append(moves, m)
	}
	body.SetFiles(moves)

	_, err = c.client.Api().V2().Location().ByUuid(dst.ID.String()).Post(ctx, body, nil)

	return err
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
	c.mu.RUnlock()

	return *c.p, nil
}
