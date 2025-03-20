package controller

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDecodeUnitVariableMap(t *testing.T) {
	t.Parallel()

	ret, err := decodeUnitVariableMap(`{"filterSubDir": "objects/attachments"}`)
	assert.NilError(t, err)
	assert.DeepEqual(t, ret, map[string]string{"filterSubDir": "objects/attachments"})

	ret, err = decodeUnitVariableMap(`{"filterSubDir": "objects/attachments"}`)
	assert.NilError(t, err)
	assert.DeepEqual(t, ret, map[string]string{"filterSubDir": "objects/attachments"})
}

func TestDecodeUnitVariableMapWithKey(t *testing.T) {
	t.Parallel()

	ret, err := decodeUnitVariableMapWithKey(`{"filterSubDir": "objects/attachments"}`, "filterSubDir")
	assert.NilError(t, err)
	assert.Equal(t, ret, "objects/attachments")

	ret, err = decodeUnitVariableMapWithKey(`{"filterSubDir": "objects/attachments"}`, "xyz")
	assert.Error(t, err, "key \"xyz\" not found")
	assert.Equal(t, ret, "")
}
