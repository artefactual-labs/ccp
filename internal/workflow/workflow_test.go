package workflow_test

import (
	"testing"

	"github.com/google/uuid"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/workflow"
)

func TestDecodeWorkflow(t *testing.T) {
	amflow, err := workflow.Default()
	assert.NilError(t, err)

	link := amflow.Links[uuid.MustParse("002716a1-ae29-4f36-98ab-0d97192669c4")]
	config := link.Config.(workflow.LinkStandardTaskConfig)
	assert.Equal(t, config.Execute, "moveSIP_v0.0")
}
