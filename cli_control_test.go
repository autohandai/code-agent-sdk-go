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

func TestBrowserHandoffAttachExactWireAndResult(t *testing.T) {
	client, requests, cleanup := newAutoresearchTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.AttachBrowserHandoff(ctx, &BrowserHandoffAttachParams{Token: "handoff-token"})
	if err != nil || !result.Success || result.SessionID == nil || *result.SessionID != "session-1" || result.WorkspaceRoot == nil || *result.WorkspaceRoot != "/workspace" || result.MessageCount == nil || *result.MessageCount != 7 {
		t.Fatalf("AttachBrowserHandoff() = %#v, %v", result, err)
	}
	assertControlRequest(t, nextControlRequest(t, requests), "autohand.browserHandoff.attach", map[string]interface{}{
		"token": "handoff-token",
	})
}

func TestBrowserHandoffAttachLatestExactWireAndResult(t *testing.T) {
	client, requests, cleanup := newAutoresearchTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.AttachLatestBrowserHandoff(ctx)
	if err != nil || result.Success || result.SessionID != nil || result.WorkspaceRoot != nil || result.MessageCount != nil {
		t.Fatalf("AttachLatestBrowserHandoff() = %#v, %v", result, err)
	}
	assertControlRequest(t, nextControlRequest(t, requests), "autohand.browserHandoff.attachLatest", map[string]interface{}{})
}

func TestAutomodeStartExactWireAndResult(t *testing.T) {
	client, requests, cleanup := newAutoresearchTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	maxIterations := 12
	completionPromise := "DONE"
	useWorktree := true
	checkpointInterval := 3
	maxRuntime := 45
	maxCost := 7.5

	result, err := client.StartAutomode(ctx, &AutomodeStartParams{
		Prompt:             "Ship the release",
		MaxIterations:      &maxIterations,
		CompletionPromise:  &completionPromise,
		UseWorktree:        &useWorktree,
		CheckpointInterval: &checkpointInterval,
		MaxRuntime:         &maxRuntime,
		MaxCost:            &maxCost,
	})
	if err != nil || !result.Success || result.SessionID == nil || *result.SessionID != "auto-1" || result.Error != nil {
		t.Fatalf("StartAutomode() = %#v, %v", result, err)
	}
	assertControlRequest(t, nextControlRequest(t, requests), "autohand.automode.start", map[string]interface{}{
		"prompt":             "Ship the release",
		"maxIterations":      float64(12),
		"completionPromise":  "DONE",
		"useWorktree":        true,
		"checkpointInterval": float64(3),
		"maxRuntime":         float64(45),
		"maxCost":            7.5,
	})
}

func TestAutomodeStatusExactWireAndResult(t *testing.T) {
	client, requests, cleanup := newAutoresearchTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.GetAutomodeStatus(ctx)
	if err != nil || !result.Active || result.Paused || result.State == nil || result.State.SessionID != "auto-1" || result.State.Status != AutomodeStatusRunning || result.State.CurrentIteration != 4 || result.State.MaxIterations != 12 || result.State.FilesCreated != 2 || result.State.FilesModified != 5 || result.State.Branch == nil || *result.State.Branch != "autohand/auto-1" || result.State.LastCheckpoint == nil || result.State.LastCheckpoint.Commit != "abc123" {
		t.Fatalf("GetAutomodeStatus() = %#v, %v", result, err)
	}
	assertControlRequest(t, nextControlRequest(t, requests), "autohand.automode.status", map[string]interface{}{})
}
