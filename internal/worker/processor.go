package worker

import (
	"context"
	"errors"
	"fmt"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
	"github.com/artefactual-labs/ccp/internal/worker/fn"
)

type processor interface {
	Do(ctx context.Context, fn string, tasks []*workerv1.Task) ([]*workerv1.TaskResult, error)
}

type processorFunc func(context.Context, []*workerv1.Task) ([]*workerv1.TaskResult, error)

type singleTaskProcessorFunc func(context.Context, *workerv1.Task) (*workerv1.TaskResult, error)

// adapter converts a singleTaskProcessorFunc into a processorFunc.
func adapter(singleTaskFunc singleTaskProcessorFunc) processorFunc {
	return func(ctx context.Context, tasks []*workerv1.Task) ([]*workerv1.TaskResult, error) {
		if len(tasks) != 1 {
			return nil, fmt.Errorf("function requires exactly one task, but received %d", len(tasks))
		}
		task := tasks[0]
		result, err := singleTaskFunc(ctx, task)
		if err != nil {
			return nil, err
		}
		return []*workerv1.TaskResult{result}, nil
	}
}

type processorImpl struct {
	funcs map[string]processorFunc
}

var _ processor = (*processorImpl)(nil)

func createProcessor() *processorImpl {
	return &processorImpl{
		funcs: map[string]processorFunc{
			"test_v0.0": adapter(fn.Test),
		},
	}
}

func (p *processorImpl) Do(ctx context.Context, fn string, tasks []*workerv1.Task) ([]*workerv1.TaskResult, error) {
	f, ok := p.funcs[fn]
	if !ok {
		return nil, errors.New("not found")
	}

	return f(ctx, tasks)
}
