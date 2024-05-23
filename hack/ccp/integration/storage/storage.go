package storage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/otiai10/copy"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/enums"
)

// I'm not making this up, it's the pipeline identifier found in mcp.sql.bz2.
var pipelineID = uuid.MustParse("dac039b9-81d1-405b-a2c9-72d7d7920c15")

type storageService struct {
	t                 testing.TB
	sharedDir         string
	transferSourceDir string
	locations         []*ssclient.Location
}

func New(t testing.TB, sharedDir, transferSourceDir string) *httptest.Server {
	storage := &storageService{
		t:                 t,
		sharedDir:         sharedDir,
		transferSourceDir: transferSourceDir,
		locations: []*ssclient.Location{
			{
				ID:        uuid.MustParse("5cbbf1f6-7abe-474e-8dda-9904083a1831"),
				URI:       "/api/v2/location/5cbbf1f6-7abe-474e-8dda-9904083a1831/",
				Purpose:   enums.LocationPurposeTS,
				Pipelines: []string{fmt.Sprintf("/api/v2/pipeline/%s/", pipelineID)},
				Path:      transferSourceDir,
			},
			{
				ID:        uuid.MustParse("df192133-3b13-4292-a219-50887d285cb3"),
				URI:       "/api/v2/location/df192133-3b13-4292-a219-50887d285cb3/",
				Purpose:   enums.LocationPurposeCP,
				Pipelines: []string{fmt.Sprintf("/api/v2/pipeline/%s/", pipelineID)},
				Path:      sharedDir,
			},
			{
				ID:        uuid.MustParse("4b3508e0-e32b-4382-ae62-1e639ddab211"),
				URI:       "/api/v2/location/4b3508e0-e32b-4382-ae62-1e639ddab211/",
				Purpose:   enums.LocationPurposeBL,
				Pipelines: []string{fmt.Sprintf("/api/v2/pipeline/%s/", pipelineID)},
				Path:      sharedDir,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v2/pipeline/{id}/", storage.readPipeline)
	mux.HandleFunc("GET /api/v2/location/{id}/", storage.readLocation)
	mux.HandleFunc("GET /api/v2/location/", storage.listLocations)
	mux.HandleFunc("GET /api/v2/location/default/{purpose}/", storage.readDefaultLocation)
	mux.HandleFunc("POST /api/v2/location/{id}/", storage.moveFiles)
	mux.HandleFunc("POST /api/v2/file/", storage.createPackage)
	mux.HandleFunc("PUT /api/v2/file/{id}/", storage.updatePackagecontents)

	srv := httptest.NewServer(mux)

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

func (s *storageService) readPipeline(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	pID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if pID != pipelineID {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}
	writeJSON(w, jPipeline{
		&ssclient.Pipeline{
			ID:  pipelineID,
			URI: fmt.Sprintf("/api/v2/pipeline/%s/", pipelineID),
		},
	})
}

func (s *storageService) readLocation(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	locationID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var match *ssclient.Location
	for _, loc := range s.locations {
		if loc.ID == locationID {
			match = loc
		}
	}
	if match == nil {
		http.Error(w, "location not found", http.StatusNotFound)
		return
	}
	writeJSON(w, jLocation{match})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "Failed to encode response.", http.StatusInternalServerError)
	}
}

func (s *storageService) listLocations(w http.ResponseWriter, req *http.Request) {
	purpose := req.URL.Query().Get("purpose")
	var matches []*ssclient.Location
	for _, item := range s.locations {
		if purpose == item.Purpose.String() {
			matches = append(matches, item)
		}
	}
	writeJSON(w, jLocationList{matches})
}

func (s *storageService) readDefaultLocation(w http.ResponseWriter, req *http.Request) {
	purpose := req.PathValue("purpose")
	var match string
	for _, item := range s.locations {
		if purpose == item.Purpose.String() {
			match = item.URI
		}
	}
	if match == "" {
		http.Error(w, "defaut location not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Location", match)
	w.WriteHeader(http.StatusFound)
}

func (s *storageService) moveFiles(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	fmt.Println(id)

	v := map[string]any{}
	if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
		http.Error(w, "cannot decode payload", http.StatusBadRequest)
		return
	}

	files, ok := v["files"]
	if !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if files, ok := files.([]interface{}); ok {
		for _, file := range files {
			if f, ok := file.(map[string]interface{}); ok {
				src := f["source"].(string)      // E.g.: "archivematica/ccp/transfer-3333975231" (relative to /home)
				dst := f["destination"].(string) // E.g.: "/var/archivematica/sharedDirectory/tmp/3323409791/Foobar/"

				src = filepath.Join(s.transferSourceDir, src)

				s.t.Logf("storage-stub: copy(%s, %s)", src, dst)
				err := copy.Copy(
					src,
					dst,
					copy.Options{},
				)
				assert.NilError(s.t, err)
			}
		}
	}
}

func (s *storageService) createPackage(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, map[string]string{"status": "OK"})
}

func (s *storageService) updatePackagecontents(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, map[string]string{"status": "OK"})
}

type jPipeline struct {
	*ssclient.Pipeline
}

func (j jPipeline) MarshalJSON() ([]byte, error) {
	type transformer struct {
		ID  uuid.UUID `json:"uuid"`
		URI string    `json:"resource_uri"`
	}
	tr := transformer{
		ID:  j.ID,
		URI: j.URI,
	}
	return json.Marshal(tr)
}

type jLocation struct {
	*ssclient.Location
}

func (j jLocation) MarshalJSON() ([]byte, error) {
	type transformer struct {
		ID           uuid.UUID             `json:"uuid"`
		URI          string                `json:"resource_uri"`
		Purpose      enums.LocationPurpose `json:"purpose"`
		Path         string                `json:"path"`
		RelativePath string                `json:"relative_path"`
		Pipelines    []string              `json:"pipeline"`
	}
	tr := transformer{
		ID:           j.ID,
		URI:          j.URI,
		Purpose:      j.Purpose,
		Path:         j.Path,
		RelativePath: j.RelativePath,
		Pipelines:    j.Pipelines,
	}
	return json.Marshal(tr)
}

type jLocationList struct {
	locations []*ssclient.Location
}

func (j jLocationList) MarshalJSON() ([]byte, error) {
	type meta struct {
		Limit    int     `json:"limit"`
		Next     *string `json:"next"`
		Offset   int     `json:"offset"`
		Previous *string `json:"previous"`
		Total    int     `json:"total_count"`
	}
	type objects []*jLocation
	type transformer struct {
		Meta    meta    `json:"meta"`
		Objects objects `json:"objects"`
	}
	tr := transformer{
		Meta: meta{
			Limit: 100,
			Total: len(j.locations),
		},
		Objects: make([]*jLocation, 0, len(j.locations)),
	}
	for _, item := range j.locations {
		tr.Objects = append(tr.Objects, &jLocation{item})
	}
	return json.Marshal(tr)
}
