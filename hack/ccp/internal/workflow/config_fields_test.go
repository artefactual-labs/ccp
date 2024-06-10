package workflow_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/enums"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/ssclientmock"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

func TestProcessingConfigForm(t *testing.T) {
	t.Parallel()

	sscli := ssclientmock.NewMockClient(gomock.NewController(t))
	sscli.EXPECT().ListLocations(mockutil.Context(), "", enums.LocationPurposeAS)
	sscli.EXPECT().ListLocations(mockutil.Context(), "", enums.LocationPurposeDS).Return(
		[]*ssclient.Location{
			{
				ID:          uuid.MustParse("2cbff4f6-cf33-4023-8df9-a1e31229a172"),
				URI:         "/api/v2/location/2cbff4f6-cf33-4023-8df9-a1e31229a172/",
				Purpose:     enums.LocationPurposeDS,
				Description: "DIPStore",
			},
		},
		nil,
	)

	wf, _ := workflow.Default()
	form := workflow.NewProcessingConfigForm(wf, sscli)
	fields, err := form.Fields(context.Background())
	assert.NilError(t, err)

	assert.Equal(t, len(fields), 27)

	// Verify sharedChainChoicesField.
	field := fields[0]
	assert.Equal(t, field.Id, "856d2d65-cd25-49fa-8da9-cabb78292894")
	assert.Equal(t, field.Name, "virus_scanning")
	assert.Equal(t, field.Label.Tx["en"], "Do you want to scan for viruses in metadata?")
	assert.Equal(t, len(field.Choice), 2)
	assert.Equal(t, len(field.Choice[0].AppliesTo), 5)
	assert.Equal(t, len(field.Choice[1].AppliesTo), 5)

	// Verify storageLocationField.
	field = fields[26]
	assert.Equal(t, field.Id, "cd844b6e-ab3c-4bc6-b34f-7103f88715de")
	assert.Equal(t, field.Name, "store_dip_location")
	assert.Equal(t, len(field.Choice), 2)
	assert.Equal(t, field.Choice[0].Label.Tx["en"], "Default location")
	assert.Equal(t, field.Choice[0].Value, "/api/v2/location/default/DS/")
	assert.Equal(t, field.Choice[1].Label.Tx["en"], "DIPStore")
	assert.Equal(t, field.Choice[1].Value, "/api/v2/location/2cbff4f6-cf33-4023-8df9-a1e31229a172/")
}
