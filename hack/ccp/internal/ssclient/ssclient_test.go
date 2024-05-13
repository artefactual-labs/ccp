package ssclient_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.artefactual.dev/tools/mockutil"
	"go.nhat.io/httpmock"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/store/fake"
)

func TestClient(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		store  func(rec *fake.MockStoreMockRecorder)
		server httpmock.Mocker
		client func(t *testing.T, c ssclient.Client)
	}{
		//
		// ReadPipeline
		//

		"ReadPipeline reads a pipeline": {
			server: httpmock.New(func(s *httpmock.Server) {
				s.ExpectGet("/api/v2/pipeline/8faae541-6124-471f-ade5-a6fe2099929d").
					ReturnHeader("Content-Type", "application/json").
					ReturnJSON(map[string]any{
						"uuid":         "8faae541-6124-471f-ade5-a6fe2099929d",
						"resource_uri": "/api/v2/pipeline/8faae541-6124-471f-ade5-a6fe2099929d",
					})
			}),
			client: func(t *testing.T, c ssclient.Client) {
				id := uuid.MustParse("8faae541-6124-471f-ade5-a6fe2099929d")

				ret, err := c.ReadPipeline(context.Background(), id)

				assert.NilError(t, err)
				assert.DeepEqual(t, ret, &ssclient.Pipeline{
					ID:  id,
					URI: "/api/v2/pipeline/8faae541-6124-471f-ade5-a6fe2099929d",
				})
			},
		},
		"ReadPipeline fails if the context is canceled": {
			client: func(t *testing.T, c ssclient.Client) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				_, err := c.ReadPipeline(ctx, uuid.Nil)

				assert.Assert(t, cmp.ErrorIs(err, context.Canceled))
			},
		},

		//
		// ReadDefaultLocation
		//

		"ReadDefaultLocation returns the default AS location": {
			store: func(rec *fake.MockStoreMockRecorder) {
				// It looks up the pipeline ID in the store.
				expectStoreReadPipelineID(rec)
			},
			server: httpmock.New(func(s *httpmock.Server) {
				// It looks up the pipeline details.
				s.ExpectGet("/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05").
					ReturnHeader("Content-Type", "application/json").
					ReturnJSON(map[string]any{
						"uuid":         "fb2b8866-6f39-4616-b6cd-fa73193a3b05",
						"resource_uri": "/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05/",
					})

				// It looks up the default location for the given purpose.
				s.ExpectGet("/api/v2/location/default/AS").
					ReturnHeader("Location", "/api/v2/location/be68cfa8-d32a-44ba-a140-2ec5d6b903e0/")

				// It looks up the location to confirm that is available to this pipeline.
				s.ExpectGet("/api/v2/location/be68cfa8-d32a-44ba-a140-2ec5d6b903e0").
					ReturnHeader("Content-Type", "application/json").
					Return(`{
  						"description": "Store AIP in standard Archivematica Directory",
  						"enabled": true,
  						"path": "/var/archivematica/sharedDirectory/www/AIPsStore",
  						"pipeline": ["/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05/"],
  						"purpose": "AS",
  						"quota": null,
  						"relative_path": "var/archivematica/sharedDirectory/www/AIPsStore",
  						"resource_uri": "/api/v2/location/be68cfa8-d32a-44ba-a140-2ec5d6b903e0/",
  						"space": "/api/v2/space/b4785c92-74c5-44d0-8d48-7f776fa55da7/",
  						"used": 0,
  						"uuid": "be68cfa8-d32a-44ba-a140-2ec5d6b903e0"
					}`)
			}),
			client: func(t *testing.T, c ssclient.Client) {
				ret, err := c.ReadDefaultLocation(context.Background(), "AS")

				assert.NilError(t, err)
				assert.DeepEqual(t, ret, &ssclient.Location{
					ID:           uuid.MustParse("be68cfa8-d32a-44ba-a140-2ec5d6b903e0"),
					URI:          "/api/v2/location/be68cfa8-d32a-44ba-a140-2ec5d6b903e0/",
					Purpose:      "AS",
					Path:         "/var/archivematica/sharedDirectory/www/AIPsStore",
					RelativePath: "var/archivematica/sharedDirectory/www/AIPsStore",
					Pipelines:    []string{"/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05/"},
				})
			},
		},

		//
		// ListLocations
		//

		"ListLocations returns a list of locations": {
			store: func(rec *fake.MockStoreMockRecorder) {
				// It looks up the pipeline ID in the store.
				expectStoreReadPipelineID(rec)
			},
			server: httpmock.New(func(s *httpmock.Server) {
				// It looks up the pipeline details.
				s.ExpectGet("/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05").
					ReturnHeader("Content-Type", "application/json").
					ReturnJSON(map[string]any{
						"uuid":         "fb2b8866-6f39-4616-b6cd-fa73193a3b05",
						"resource_uri": "/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05/",
					})

				// It looks up the location list endpoint.
				s.ExpectGet("/api/v2/location?limit=100&pipeline__uuid=fb2b8866-6f39-4616-b6cd-fa73193a3b05&purpose=DS").
					ReturnHeader("Content-Type", "application/json").
					Return(`{
						"meta": {
							"limit": 100,
							"next": null,
							"offset": 0,
							"previous": null,
							"total_count": 1
						},
						"objects": [
							{
								"description": "Store DIP in standard Archivematica Directory",
								"enabled": true,
								"path": "/var/archivematica/sharedDirectory/www/DIPsStore",
								"pipeline": ["/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05/"],
								"purpose": "DS",
								"quota": null,
								"relative_path": "var/archivematica/sharedDirectory/www/DIPsStore",
								"resource_uri": "/api/v2/location/18d6c0c4-afcd-4ee5-a9b0-19158cb199af/",
								"space": "/api/v2/space/b4785c92-74c5-44d0-8d48-7f776fa55da7/",
								"used": 0,
								"uuid": "18d6c0c4-afcd-4ee5-a9b0-19158cb199af"
							}
						]
					}`)
			}),
			client: func(t *testing.T, c ssclient.Client) {
				ret, err := c.ListLocations(context.Background(), "", "DS")

				assert.NilError(t, err)
				assert.DeepEqual(t, ret, []*ssclient.Location{
					{
						ID:           uuid.MustParse("18d6c0c4-afcd-4ee5-a9b0-19158cb199af"),
						URI:          "/api/v2/location/18d6c0c4-afcd-4ee5-a9b0-19158cb199af/",
						Purpose:      "DS",
						Path:         "/var/archivematica/sharedDirectory/www/DIPsStore",
						RelativePath: "var/archivematica/sharedDirectory/www/DIPsStore",
						Pipelines:    []string{"/api/v2/pipeline/fb2b8866-6f39-4616-b6cd-fa73193a3b05/"},
					},
				})
			},
		},

		//
		// CopyFiles
		//

		"CopyFiles ...": { // TODO
			server: nil,
			client: func(t *testing.T, c ssclient.Client) {},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store := fake.NewMockStore(gomock.NewController(t))
			if tc.store != nil {
				tc.store(store.EXPECT())
			}

			var srv *httpmock.Server
			if tc.server == nil {
				srv = httpmock.NewServer().WithTest(t)
			} else {
				srv = tc.server(t)
			}

			config := ssclient.Config{srv.URL(), "username", "api-key"}
			c, err := ssclient.NewClient(nil, store, config)
			assert.NilError(t, err)

			tc.client(t, c)
		})
	}
}

func expectStoreReadPipelineID(rec *fake.MockStoreMockRecorder) {
	rec.
		ReadPipelineID(mockutil.Context()).
		Return(uuid.MustParse("fb2b8866-6f39-4616-b6cd-fa73193a3b05"), nil).
		Times(1)
}
