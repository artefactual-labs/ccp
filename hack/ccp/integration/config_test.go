package integration_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"golang.org/x/exp/slices"
)

const prefix = "CCP_INTEGRATION"

var (
	enabled        = getEnvBool("ENABLED", "no")
	useCompose     = getEnvBool("USE_COMPOSE", "no")
	useStdout      = getEnvBool("USE_STDOUT", "yes")
	transferSource = getEnvRequired("TRANSFER_SOURCE")
)

func getEnv(name, fallback string) string {
	v := os.Getenv(fmt.Sprintf("%s_%s", prefix, name))
	if v == "" {
		return fallback
	}
	return v
}

func getEnvRequired(name string) string {
	v := getEnv(name, "")
	if v == "" {
		log.Fatalf("Required env %s_%s is empty.", prefix, name)
	}
	return v
}

func getEnvBool(name, fallback string) bool {
	if v := getEnv(name, fallback); slices.Contains([]string{"yes", "1", "on", "true"}, v) {
		return true
	} else {
		return false
	}
}

func requireFlag(t *testing.T) {
	if !enabled {
		t.Skip("Skipping integration tests (CCP_INTEGRATION_ENABLED=no).")
	}
}
