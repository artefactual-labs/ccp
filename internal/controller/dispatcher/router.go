package dispatcher

import (
	"fmt"

	"github.com/artefactual-labs/ccp/internal/workflow"
)

// backendRouter is the function signature for routing tasks to the appropriate
// backend.
type backendRouter func(backends map[string]Backend, linkConfig *workflow.LinkStandardTaskConfig) Backend

// DefaultBackendRouter defaults to the gearmin backend (MCPClient) but promotes
// alternative backends when available.
func DefaultBackendRouter(backends map[string]Backend, linkConfig *workflow.LinkStandardTaskConfig) Backend {
	// This is where we list the client scripts that we want to route to the
	// Connect backend. The full list of client scripts can be obtained by
	// running:
	//   jq -r '[.. | objects | .execute? | select(. != null)] | unique | .[]' workflow.json
	rules := map[string]bool{
		"test_v0.0": true,
	}

	// Determine whether the script has been refactored.
	var refactored bool
	name := linkConfig.Execute
	if rule, ok := rules[name]; ok {
		if rule {
			refactored = true
		}
	}

	// If refactored, use the connect backend.
	backendID := gearminBackendID
	if refactored {
		backendID = workhubBackendID
	}

	// Panic if the backend is not found. This should never happen.
	if be, ok := backends[backendID]; !ok {
		panic(fmt.Sprintf("backend %s not found", backendID))
	} else {
		return be
	}
}
