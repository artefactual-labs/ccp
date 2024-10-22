package controller

import (
	"context"
	"fmt"

	"github.com/elliotchance/orderedmap/v2"

	"github.com/artefactual-labs/ccp/internal/python"
	"github.com/artefactual-labs/ccp/internal/workflow"
)

// A chain is used for passing information between jobs.
//
// In Archivematica the workflow is structured around chains and links.
// A chain is a sequence of links used to accomplish a broader task or set of
// tasks, carrying local state relevant only for the duration of the chain.
// The output of a chain is placed in a watched directory to trigger the next
// chain.
//
// In MCPServer, `chain.jobChain` is implemented as an iterator, simplifying
// the process of moving through the jobs in a chain. When a chain completes,
// the queue manager checks the queues for ay work awaiting to be processed,
// which could be related to other packages.
//
// In a3m, chains and watched directories were removed, but it's hard to do it
// without introducing backward-incompatible changes given the reliance on it
// in some edge cases like reingest, etc.
type chain struct {
	// The properties of the chain as described by the workflow document.
	wc *workflow.Chain

	// A map of replacement variables for tasks.
	// TODO: why are we not using replacementMappings instead?
	context packageContext
}

type packageContext struct {
	*orderedmap.OrderedMap[string, string]
}

func newChain(wc *workflow.Chain) *chain {
	return &chain{
		wc:      wc,
		context: packageContext{orderedmap.NewOrderedMap[string, string]()},
	}
}

// update the context of the chain with a new map.
func (c *chain) update(kvs map[string]string) {
	for k, v := range kvs {
		c.context.Set(k, string(v))
	}
}

// load the database package context into this chain.
//
// TODO: we shouldn't need one UnitVariable per chain, with all the same values.
func (c *chain) load(ctx context.Context, pkg *Package) error {
	vars, err := pkg.store.ReadUnitVars(ctx, pkg.id, "", "replacementDict")
	if err != nil {
		return err
	}

	for _, item := range vars {
		if item.Value == nil {
			continue
		}
		m, err := python.EvalMap(*item.Value)
		if err != nil {
			pkg.logger.Error(err, "Failed to eval unit variable value %q.", *item.Value)
			continue
		}
		for k, v := range m {
			c.context.Set(k, v)
		}
	}

	kvs := []any{"len", c.context.Len()}
	for el := c.context.Front(); el != nil; el = el.Next() {
		kvs = append(kvs, fmt.Sprintf("var:%s", el.Key), el.Value)
	}
	pkg.logger.V(2).Info("Package context loaded from the database.", kvs...)

	return nil
}
