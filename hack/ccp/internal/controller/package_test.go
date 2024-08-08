package controller

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestReplacements(t *testing.T) {
	t.Parallel()

	t.Run("Updates itself with a given packageContext", func(t *testing.T) {
		t.Parallel()

		c := newChain(nil)
		c.context.Set("%path%", "/mnt/disk")
		c.context.Set("%name%", `Dr. Evelyn "The Innovator" O'Neill: The Complete Digital Archives`)

		rm := replacementMapping(map[string]replacement{
			"%uuid%": "91354225-f28b-433c-8280-cf6a5edea2ff",
			"%job%":  `cool \\stuff`,
		}).update(c)

		assert.Equal(t,
			rm.replaceValues(`%name% with path="%path%" and uuid="%uuid%" did: %job%`),
			`Dr. Evelyn \"The Innovator\" O'Neill: The Complete Digital Archives with path="/mnt/disk" and uuid="91354225-f28b-433c-8280-cf6a5edea2ff" did: cool \\\\\\\\stuff`,
		)
	})
}

func TestCopyTransfer(t *testing.T) {
	t.Parallel()
}
