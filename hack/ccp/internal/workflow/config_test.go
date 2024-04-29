package workflow_test

import (
	"os"
	"testing"

	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
	"gotest.tools/v3/assert"
)

func TestParseConfig(t *testing.T) {
	f, err := os.Open("../../hack/processingMCP.xml")
	assert.NilError(t, err)
	t.Cleanup(func() { f.Close() })

	config, err := workflow.ParseConfig(f)
	assert.NilError(t, err)
	assert.Equal(t, len(config), 32)
}
