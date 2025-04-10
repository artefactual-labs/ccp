package provisioner

import (
	"fmt"

	"github.com/google/uuid"
)

// IdentifierGenerator provides stable identifiers for workers.
type IdentifierGenerator interface {
	// Yield returns a stable identifier for the given worker index.
	Yield(index int) uuid.UUID
}

// uuidGenerator implements IdentifierGenerator using UUIDv5.
type uuidGenerator struct {
	namespace uuid.UUID
}

var _ IdentifierGenerator = (*uuidGenerator)(nil)

var defaultUUIDGeneratorNamespace = uuid.MustParse("f8a6b7a5-9b7c-4e0a-8d2f-1b9e8a7b6c5d")

// NewUUIDGenerator creates a new generator using the default namespace.
func NewUUIDGenerator() *uuidGenerator {
	return &uuidGenerator{namespace: defaultUUIDGeneratorNamespace}
}

// Yield generates a UUIDv5 based on the namespace and the worker index.
func (g *uuidGenerator) Yield(index int) uuid.UUID {
	name := fmt.Sprintf("worker-%d", index)
	return uuid.NewSHA1(g.namespace, []byte(name))
}
