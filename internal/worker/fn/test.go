package fn

import (
	"context"
	"os"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
)

func Test(ctx context.Context, task *workerv1.Task) (*workerv1.TaskResult, error) {
	exitCode, err := test(ctx, task.Arguments)
	if err != nil {
		return nil, err
	}

	res := &workerv1.TaskResult{
		Id:         task.Id,
		Stdout:     nil,
		Stderr:     nil,
		FinishedAt: timestamppb.Now(),
		ExitCode:   int32(exitCode), //nolint: gosec
	}

	return res, nil
}

// 2 = error, 1 = not a dir, 0 = a dir
func test(ctx context.Context, args string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 2, err
	}
	if args == "" {
		return 2, nil
	}
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] != "-d" {
		return 2, nil
	}
	path := parts[1]
	info, err := os.Stat(path)
	if err != nil {
		return 2, nil
	}
	if info.IsDir() {
		return 0, nil
	}
	return 1, nil
}
