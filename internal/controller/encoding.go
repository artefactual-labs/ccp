package controller

import (
	"encoding/json"
	"fmt"
)

func decodeUnitVariableMap(value string) (map[string]string, error) {
	m := map[string]string{}
	if err := json.Unmarshal([]byte(value), &m); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}

	return m, nil
}

func decodeUnitVariableMapWithKey(value, key string) (string, error) {
	m, err := decodeUnitVariableMap(value)
	if err != nil {
		return "", err
	}

	ret, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %q not found", key)
	}

	return ret, nil
}
