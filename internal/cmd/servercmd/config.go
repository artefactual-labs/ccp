package servercmd

import (
	"io"

	"github.com/artefactual-labs/ccp/internal/api/admin"
	"github.com/artefactual-labs/ccp/internal/cmd/rootcmd"
	"github.com/artefactual-labs/ccp/internal/cmd/servercmd/metrics"
	"github.com/artefactual-labs/ccp/internal/webui"
)

type Config struct {
	rootConfig *rootcmd.Config
	out        io.Writer
	sharedDir  string
	workflow   string
	db         databaseConfig
	api        apiConfig
	gearmin    gearminConfig
	webui      webui.Config
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
