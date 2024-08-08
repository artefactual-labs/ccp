package workflow_test

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

func TestProcessingConfigForm(t *testing.T) {
	t.Parallel()

	wf, _ := workflow.Default()
	form := workflow.NewProcessingConfigForm(wf)
	fields, err := form.Fields(context.Background())
	assert.NilError(t, err)

	assert.Equal(t, len(fields), 25)

	// Verify sharedChainChoicesField.
	field := fields[0]
	assert.Equal(t, field.Id, "856d2d65-cd25-49fa-8da9-cabb78292894")
	assert.Equal(t, field.Name, "virus_scanning")
	assert.Equal(t, field.Label.Tx["en"], "Do you want to scan for viruses in metadata?")
	assert.Equal(t, len(field.Choice), 2)
	assert.Equal(t, len(field.Choice[0].AppliesTo), 5)
	assert.Equal(t, len(field.Choice[1].AppliesTo), 5)
}
