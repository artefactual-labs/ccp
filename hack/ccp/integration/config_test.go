package integration_test

import "slices"

// automatedConfigTransformations is the preferred list of transformations that
// we apply to the "automated" config in our tests.
var automatedConfigTransformations = []string{
	// Send SIP to backlog.
	"bb194013-597c-4e4a-8493-b36d190f8717", "7065d256-2f47-4b7d-baec-2c4699626121",
	// Virus scanning disabled.
	"856d2d65-cd25-49fa-8da9-cabb78292894", "63767e4b-9ce8-4fe2-8724-65cc1f763de0",
	"1dad74a2-95df-4825-bbba-dca8b91d2371", "697c0883-798d-4af7-b8b6-101c7f709cd5",
	"7e81f94e-6441-4430-a12d-76df09181b66", "77355172-b437-4324-9dcc-e2607ad27cb1",
	"390d6507-5029-4dae-bcd4-ce7178c9b560", "63be6081-bee8-4cf5-a453-91893e31940f",
	"97a5ddc0-d4e0-43ac-a571-9722405a0a9b", "7f5244fe-590b-4e38-beaf-0cf1ccb9e71b",
}

func configTransformations(processingConfigTransformations ...string) []string {
	return slices.Concat(
		automatedConfigTransformations,
		processingConfigTransformations,
	)
}
