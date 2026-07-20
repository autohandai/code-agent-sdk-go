package autohand

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func nextControlRequest(t *testing.T, requests <-chan capturedRPCRequest) capturedRPCRequest {
	t.Helper()
	select {
	case request := <-requests:
		return request
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for control request")
		return capturedRPCRequest{}
	}
}

func assertControlRequest(t *testing.T, request capturedRPCRequest, method string, params map[string]interface{}) {
	t.Helper()
	if request.Method != method {
		t.Fatalf("method = %q, want %q", request.Method, method)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(request.Params, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, params) {
		t.Fatalf("params = %#v, want %#v", got, params)
	}
}

func TestResetExactWireAndResult(t *testing.T) {
	client, requests, cleanup := newAutoresearchTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.Reset(ctx)
	if err != nil || result.SessionID != "session-new" {
		t.Fatalf("Reset() = %#v, %v", result, err)
	}
	assertControlRequest(t, nextControlRequest(t, requests), "autohand.reset", map[string]interface{}{})
}
