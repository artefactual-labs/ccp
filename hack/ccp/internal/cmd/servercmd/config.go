package servercmd

import (
	"io"

	"github.com/artefactual/archivematica/hack/ccp/internal/api/admin"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/rootcmd"
	"github.com/artefactual/archivematica/hack/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual/archivematica/hack/ccp/internal/shim"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/webui"
)

type Config struct {
	rootConfig *rootcmd.Config
	out        io.Writer
	sharedDir  string
	workflow   string
	db         databaseConfig
	api        apiConfig
	gearmin    gearminConfig
	ssclient   ssclient.Config
	webui      webui.Config
	shim       shim.Config
	metrics    metrics.Config
}

type databaseConfig struct {
	driver string
	dsn    string
}

type apiConfig struct {
	admin admin.Config
}

type gearminConfig struct {
	addr string
}
