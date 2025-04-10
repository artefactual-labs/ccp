package workhub_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	connect_validate "connectrpc.com/validate"
	"github.com/go-logr/logr/testr"
	"github.com/google/uuid"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	workerv1 "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1"
	workerv1connect "github.com/artefactual-labs/ccp/internal/api/gen/archivematica/ccp/worker/v1beta1/workerv1beta1connect"
	"github.com/artefactual-labs/ccp/internal/workhub"
)

func TestServer(t *testing.T) {
	t.Parallel()

	t.Run("GrabBatch rejects invalid GrabBatchRequest", func(t *testing.T) {
		t.Parallel()

		_, _, client := setUpTestServer(t)

		req := connect.NewRequest(&workerv1.GrabBatchRequest{})
		resp, err := client.GrabBatch(t.Context(), req)

		assert.Assert(t, resp == nil)

		var connectError *connect.Error
		assert.Assert(t, errors.As(err, &connectError))
		assert.Equal(t, connectError.Code(), connect.CodeInvalidArgument)
		checkConnectError(t, connectError, []violationSpec{
			{
				ruleID:    "string.uuid_empty",
				fieldPath: "worker_id",
				message:   "value is empty, which is not a valid UUID",
			},
		})
	})

	t.Run("GrabBatch times out when no batch is available", func(t *testing.T) {
		t.Parallel()

		_, _, client := setUpTestServer(t)
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		req := connect.NewRequest(&workerv1.GrabBatchRequest{WorkerId: uuid.NewString()})
		resp, err := client.GrabBatch(ctx, req)

		assert.Assert(t, resp == nil)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("Basic produce-consumer flow", func(t *testing.T) {
		ctx := t.Context()
		workerID := uuid.New()
		hub, _, client := setUpTestServer(t)

		// Sample tasks for the batch
		task1ID := uuid.New().String()
		task2ID := uuid.New().String()
		batchToEnqueue := []*workerv1.Task{
			{
				Id:        task1ID,
				Function:  "fn1",
				Arguments: "",
			},
			{
				Id:        task2ID,
				Function:  "fn1",
				Arguments: "",
			},
		}

		// Producer: Enqueue the batch
		pendingBatch, err := hub.Enqueue(ctx, "fn1", batchToEnqueue)
		assert.NilError(t, err)
		t.Log(pendingBatch, workerID, client)

		// Producer: Start waiting for results in a goroutine
		var wg sync.WaitGroup
		var receivedResults workhub.BatchResult
		var waitErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Use a reasonable timeout for the wait
			waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Second)
			defer waitCancel()
			receivedResults, waitErr = pendingBatch.Wait(waitCtx)
		}()

		// Worker: Grab the batch
		grabReq := connect.NewRequest(&workerv1.GrabBatchRequest{
			WorkerId: workerID.String(),
		})
		grabResp, err := client.GrabBatch(ctx, grabReq)
		assert.NilError(t, err)
		assert.Equal(t, len(grabResp.Msg.Tasks), 2)

		/*
			// Assertions: Verify grabbed batch details
			assert.Equal(t, pendingBatch.ID.String(), grabResp.Msg.Id, "Grabbed batch ID should match enqueued batch ID")
			// Assign batch ID to enqueued tasks for comparison (as Hub assigns it internally)
			for _, task := range batchToEnqueue {
				task.BatchId = grabResp.Msg.Id
			}
			assert.ElementsMatch(t, batchToEnqueue, grabResp.Msg.Tasks, "Grabbed tasks should match enqueued tasks")
		*/

		// Worker: Complete the batch
		resultsToSend := []*workerv1.TaskResult{
			{Id: task1ID, ExitCode: 0, Stdout: []byte("output1"), FinishedAt: timestamppb.Now()},
			{Id: task2ID, ExitCode: 1, Stderr: []byte("error2"), FinishedAt: timestamppb.Now()},
		}
		completeReq := connect.NewRequest(&workerv1.CompleteBatchRequest{
			WorkerId: workerID.String(),
			BatchId:  grabResp.Msg.Id,
			Results:  resultsToSend,
		})
		_, err = client.CompleteBatch(ctx, completeReq)
		assert.NilError(t, err)

		// Producer: Wait for the Wait call to finish
		wg.Wait()

		assert.NilError(t, waitErr)
		assert.DeepEqual(t,
			receivedResults,
			workhub.BatchResult{
				task1ID: {Id: task1ID, ExitCode: 0, Stdout: []byte("output1"), FinishedAt: timestamppb.Now()},
				task2ID: {Id: task2ID, ExitCode: 1, Stderr: []byte("error2"), FinishedAt: timestamppb.Now()},
			},
			protocmp.Transform(),
			protocmp.IgnoreFields(&workerv1.TaskResult{}, "finished_at"),
		)
	})

	t.Run("Wait_Timeout", func(t *testing.T) {
		// TODO: Implement test case
		// 1. Enqueue a batch using the hub directly.
		// 2. Call Wait on the pending batch with a very short timeout context.
		// 3. Assert the error is context.DeadlineExceeded.
		t.Skip("TODO: Implement Wait_Timeout")
	})
}

func setUpTestServer(t *testing.T) (*workhub.Hub, *httptest.Server, workerv1connect.WorkerServiceClient) { //nolint:unparam
	t.Helper()

	logger := testr.New(t)
	hub := workhub.NewHub(logger)
	t.Cleanup(func() { hub.Close() })
	mux := http.NewServeMux()

	validator, err := connect_validate.NewInterceptor()
	assert.NilError(t, err)

	path, handler := workerv1connect.NewWorkerServiceHandler(hub, connect.WithInterceptors(validator))
	mux.Handle(path, handler)

	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(func() { server.Close() })

	client := workerv1connect.NewWorkerServiceClient(
		server.Client(),
		server.URL,
	)

	return hub, server, client
}

// violationSpec is a simple representation of fields tested when inspecting
// a connect.Error that we expect to contain Protovalidate validate.Violations
// messages.
type violationSpec struct {
	ruleID    string
	fieldPath string
	message   string
}

// checkConnectError receives a connect.Error that's expected to be returned
// by the connectrpc.com/validate validation intercepter. It's expected to
// contain one ErrorDetail, and for the value of that ErrorDetail to be a
// validate.Violations message.
//
// It uses a slice of violationSpec as an expectation for the contents of
// validate.Violations.GetViolations().
func checkConnectError(t testing.TB, connectError *connect.Error, specs []violationSpec) {
	t.Helper()

	details := connectError.Details()
	assert.Equal(t, len(details), 1, "Connect error had %d details instead of one.", len(details))

	detail, err := details[0].Value()
	assert.NilError(t, err, "Couldn't get value of first error detail: %v.", err)

	violations, ok := detail.(*validate.Violations)
	assert.Assert(t, ok, "Received unexpected type of detail: %v.", detail)
	assert.Assert(t, specs != nil, "Received a connect error with Violations but no spec was provided.")

	allViolations := violations.GetViolations()
	assert.Assert(t, len(allViolations) == len(specs), "Violations returned %d violations instead of %d.", len(allViolations), len(specs))

	for i, spec := range specs {
		violation := allViolations[i]
		assert.Equal(t, violation.GetRuleId(), spec.ruleID, "Wrong ruleID.")
		fieldPath := protovalidate.FieldPathString(violation.GetField())
		assert.Equal(t, fieldPath, spec.fieldPath, "Wrong fieldPath.")
		assert.Equal(t, violation.GetMessage(), spec.message, "Wrong message.")
	}
}
