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

func TestBrowserHandoffCreateExactWireAndResult(t *testing.T) {
	client, requests, cleanup := newAutoresearchTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	extensionID := "ext-id"
	installURL := "https://example.test/install"

	result, err := client.CreateBrowserHandoff(ctx, &BrowserHandoffCreateParams{
		ExtensionID: &extensionID,
		InstallURL:  &installURL,
	})
	if err != nil || result.Token != "handoff-token" || result.SessionID != "session-1" || result.WorkspaceRoot != "/workspace" || result.CreatedAt != "2026-07-20T01:00:00Z" || result.ExpiresAt != "2026-07-20T01:05:00Z" || result.URL != "chrome-extension://ext/continue" {
		t.Fatalf("CreateBrowserHandoff() = %#v, %v", result, err)
	}
	assertControlRequest(t, nextControlRequest(t, requests), "autohand.browserHandoff.create", map[string]interface{}{
		"extensionId": "ext-id",
		"installUrl":  "https://example.test/install",
	})
}
