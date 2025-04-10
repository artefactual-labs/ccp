package dispatcher_test

import (
	"testing"

	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/ccp/internal/controller/dispatcher"
	"github.com/artefactual-labs/ccp/internal/controller/dispatcher/dispatchermock"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

func TestDefaultBackendRouter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		execute         string
		expectedBackend string
	}{
		{"Routes test_v0.0 to workhub", "test_v0.0", "workhub"},
		{"Routes unknown script to gearmin", "unknown_script", "gearmin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gearminBackend := dispatchermock.NewMockBackend(ctrl)
			workhubBackend := dispatchermock.NewMockBackend(ctrl)
			backends := map[string]dispatcher.Backend{
				"gearmin": gearminBackend,
				"workhub": workhubBackend,
			}

			gearminBackend.EXPECT().Name().Return("gearmin").AnyTimes()
			workhubBackend.EXPECT().Name().Return("workhub").AnyTimes()

			linkConfig := &workflow.LinkStandardTaskConfig{Execute: tt.execute}
			backend := dispatcher.DefaultBackendRouter(backends, linkConfig)
			assert.Equal(t, tt.expectedBackend, backend.Name())
		})
	}
}
