package workflow_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

func TestInstallBuiltinConfigs(t *testing.T) {
	tmpDir := fs.NewDir(t, "")

	err := workflow.InstallBuiltinConfigs(tmpDir.Path())
	assert.NilError(t, err)

	choices, err := workflow.ParseConfigFile(tmpDir.Join("defaultProcessingMCP.xml"))
	assert.NilError(t, err)
	assert.Equal(t, len(choices), 17, "unexpected number of preconfigured choices found in the default config")

	choices, err = workflow.ParseConfigFile(tmpDir.Join("automatedProcessingMCP.xml"))
	assert.NilError(t, err)
	assert.Equal(t, len(choices), 32, "unexpected number of preconfigured choices found in the automated config")

	expected := fs.Expected(t,
		fs.WithFile("automatedProcessingMCP.xml", "", fs.MatchAnyFileContent, fs.MatchAnyFileMode),
		fs.WithFile("defaultProcessingMCP.xml", "", fs.MatchAnyFileContent, fs.MatchAnyFileMode),
	)
	assert.Assert(t, fs.Equal(tmpDir.Path(), expected))
}
