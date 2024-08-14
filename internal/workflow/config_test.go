package workflow_test

import (
	"testing"

	"github.com/google/uuid"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/ccp/internal/workflow"
)

func TestParseConfigFile(t *testing.T) {
	choices, err := workflow.ParseConfigFile("../../hack/helpers/processingMCP.xml")
	assert.NilError(t, err)

	assert.Equal(t, len(choices), 32, "unexpected number of preconfigured choices found")

	assert.Equal(t, choices[0].Comment, "") // TODO: we're not preserving the comment yet.
	assert.Equal(t, choices[0].LinkID(), uuid.MustParse("5e58066d-e113-4383-b20b-f301ed4d751c"))
	assert.Equal(t, choices[0].ChainID(), uuid.MustParse("8d29eb3d-a8a8-4347-806e-3d8227ed44a1"))
	assert.Equal(t, choices[0].Value(), "8d29eb3d-a8a8-4347-806e-3d8227ed44a1")
}

func TestSaveConfigFile(t *testing.T) {
	dir := fs.NewDir(t, "")

	err := workflow.SaveConfigFile(dir.Join("processingMCP.xml"), []workflow.Choice{
		{
			Comment:   "Store DIP",
			AppliesTo: "5e58066d-e113-4383-b20b-f301ed4d751c",
			GoToChain: "8d29eb3d-a8a8-4347-806e-3d8227ed44a1",
		},
	})
	assert.NilError(t, err)

	expected := fs.Expected(t,
		fs.WithFile("processingMCP.xml", `<processingMCP>
  <preconfiguredChoices>
    <!-- Store DIP -->
    <preconfiguredChoice>
      <appliesTo>5e58066d-e113-4383-b20b-f301ed4d751c</appliesTo>
      <goToChain>8d29eb3d-a8a8-4347-806e-3d8227ed44a1</goToChain>
    </preconfiguredChoice>
  </preconfiguredChoices>
</processingMCP>`),
	)
	assert.Assert(t, fs.Equal(dir.Path(), expected))
}
