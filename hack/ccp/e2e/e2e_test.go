package e2e

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestE2E(t *testing.T) {
	t.Log("Hello world!")

	entries, err := os.ReadDir("/var/archivematica/sharedDirectory")
	assert.NilError(t, err)
	for _, entry := range entries {
		t.Log(entry)
	}

	t.Log("Empty?")
}
