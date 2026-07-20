package autohand

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"testing"
	"time"
)

func TestAutoresearchRPCMethods(t *testing.T) {
	client, requests, closeTransport := newAutoresearchTestClient(t)
	defer closeTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start, err := client.StartAutoresearch(ctx, &AutoresearchStartParams{
		Objective:      "Reduce latency",
		MetricName:     "p95_ms",
		MetricUnit:     "ms",
		Direction:      AutoresearchLower,
		MeasureCommand: "go test ./...",
	})
	if err != nil {
		t.Fatalf("StartAutoresearch: %v", err)
	}
	if !start.Success || start.Instruction != "Run the next experiment" {
		t.Fatalf("unexpected start result: %+v", start)
	}

	status, err := client.GetAutoresearchStatus(ctx)
	if err != nil {
		t.Fatalf("GetAutoresearchStatus: %v", err)
	}
	if !status.Active || status.RunsLogged != 2 {
		t.Fatalf("unexpected status result: %+v", status)
	}

	if _, err := client.StopAutoresearch(ctx); err != nil {
		t.Fatalf("StopAutoresearch: %v", err)
	}
	if _, err := client.GetAutoresearchHistory(ctx); err != nil {
		t.Fatalf("GetAutoresearchHistory: %v", err)
	}
	if _, err := client.ReplayAutoresearch(ctx, &AutoresearchReplayParams{AttemptID: "attempt-1", Evaluator: AutoresearchEvaluatorOriginal}); err != nil {
		t.Fatalf("ReplayAutoresearch: %v", err)
	}
	if _, err := client.RescoreAutoresearch(ctx, AutoresearchRescoreAttempt("attempt-1")); err != nil {
		t.Fatalf("RescoreAutoresearch: %v", err)
	}
	if _, err := client.CompareAutoresearch(ctx, &AutoresearchCompareParams{LeftAttemptID: "attempt-1", RightAttemptID: "attempt-2"}); err != nil {
		t.Fatalf("CompareAutoresearch: %v", err)
	}
	if _, err := client.GetAutoresearchPareto(ctx); err != nil {
		t.Fatalf("GetAutoresearchPareto: %v", err)
	}
	if _, err := client.PinAutoresearch(ctx, &AutoresearchPinParams{AttemptID: "attempt-1", Pinned: true}); err != nil {
		t.Fatalf("PinAutoresearch: %v", err)
	}
	if _, err := client.PruneAutoresearch(ctx, &AutoresearchPruneParams{DryRun: boolPointer(true)}); err != nil {
		t.Fatalf("PruneAutoresearch: %v", err)
	}

	wantMethods := []string{
		"autohand.autoresearch.start",
		"autohand.autoresearch.status",
		"autohand.autoresearch.stop",
		"autohand.autoresearch.history",
		"autohand.autoresearch.replay",
		"autohand.autoresearch.rescore",
		"autohand.autoresearch.compare",
		"autohand.autoresearch.pareto",
		"autohand.autoresearch.pin",
		"autohand.autoresearch.prune",
	}
	gotRequests := make([]capturedRPCRequest, 0, len(wantMethods))
	for range wantMethods {
		gotRequests = append(gotRequests, <-requests)
	}
	gotMethods := make([]string, 0, len(gotRequests))
	for _, request := range gotRequests {
		gotMethods = append(gotMethods, request.Method)
	}
	if !reflect.DeepEqual(gotMethods, wantMethods) {
		t.Fatalf("RPC methods = %v, want %v", gotMethods, wantMethods)
	}
	var startParams map[string]interface{}
	if err := json.Unmarshal(gotRequests[0].Params, &startParams); err != nil {
		t.Fatalf("unmarshal start params: %v", err)
	}
	if startParams["objective"] != "Reduce latency" || startParams["direction"] != "lower" {
		t.Fatalf("unexpected start params: %v", startParams)
	}
}

func TestAutoresearchNotificationsMapToTypedEvents(t *testing.T) {
	client := NewRPCClient(&Config{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := client.Events(ctx)
	client.transport.handleLine(`{"jsonrpc":"2.0","method":"autohand.autoresearch.status","params":{"active":true,"goal":"Reduce latency","iteration":2,"maxIterations":8,"runsLogged":3,"statusText":"Auto-research active","subcommand":"status","timestamp":"2026-07-17T00:00:00Z"}}`)
	client.transport.handleLine(`{"jsonrpc":"2.0","method":"autohand.autoresearch.event","params":{"operation":"replay","phase":"completed","attemptId":"attempt-1","success":true,"timestamp":"2026-07-17T00:00:01Z"}}`)

	first := <-events
	lifecycle, ok := first.(AutoresearchLifecycleEvent)
	if !ok || lifecycle.Phase != AutoresearchPhaseStatus || lifecycle.RunsLogged != 3 {
		t.Fatalf("unexpected lifecycle event: %#v", first)
	}
	second := <-events
	operation, ok := second.(AutoresearchOperationEvent)
	if !ok || operation.Operation != AutoresearchOperationReplay || operation.Phase != AutoresearchOperationCompleted || operation.AttemptID != "attempt-1" {
		t.Fatalf("unexpected operation event: %#v", second)
	}
}

func TestAutoresearchRescoreParamsRejectInvalidStates(t *testing.T) {
	invalid := []*AutoresearchRescoreParams{
		{},
		{AttemptID: "attempt-1", All: true},
	}
	for _, params := range invalid {
		if err := params.Validate(); err == nil {
			t.Fatalf("Validate(%+v) unexpectedly succeeded", params)
		}
	}
	if err := AutoresearchRescoreAll().Validate(); err != nil {
		t.Fatalf("AutoresearchRescoreAll.Validate: %v", err)
	}
}

func TestAutoresearchHookEventConstants(t *testing.T) {
	want := map[HookEvent]string{
		HookEventAutoresearchStart:    "autoresearch:start",
		HookEventAutoresearchDecision: "autoresearch:decision",
		HookEventAutoresearchReplay:   "autoresearch:replay",
		HookEventAutoresearchRescore:  "autoresearch:rescore",
		HookEventAutoresearchPrune:    "autoresearch:prune",
		HookEventAutoresearchComplete: "autoresearch:complete",
		HookEventAutoresearchError:    "autoresearch:error",
	}
	for event, expected := range want {
		if string(event) != expected {
			t.Errorf("hook event %q = %q, want %q", event, event, expected)
		}
	}
}

func TestTransportRequestIDsStartAtOne(t *testing.T) {
	transport := NewTransport(&Config{})
	if transport.nextID != 1 {
		t.Fatalf("NewTransport nextID = %d, want 1", transport.nextID)
	}
}

type capturedRPCRequest struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func newAutoresearchTestClient(t *testing.T) (*RPCClient, <-chan capturedRPCRequest, func()) {
	t.Helper()
	reader, writer := io.Pipe()
	transport := &Transport{
		stdin:     writer,
		callbacks: make(map[int]chan transportResponse),
		notify:    make(map[string]func(json.RawMessage)),
		nextID:    1,
		timeout:   2 * time.Second,
	}
	client := &RPCClient{transport: transport}
	client.setupNotifications()
	requests := make(chan capturedRPCRequest, 16)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var request capturedRPCRequest
			if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
				return
			}
			requests <- request
			result := map[string]interface{}{"success": true}
			var responseResult interface{} = result
			switch request.Method {
			case "autohand.reset":
				responseResult = map[string]interface{}{"sessionId": "session-new"}
			case "autohand.browserHandoff.create":
				responseResult = map[string]interface{}{
					"token": "handoff-token", "sessionId": "session-1", "workspaceRoot": "/workspace",
					"createdAt": "2026-07-20T01:00:00Z", "expiresAt": "2026-07-20T01:05:00Z", "url": "chrome-extension://ext/continue",
				}
			case "autohand.browserHandoff.attach":
				responseResult = map[string]interface{}{"success": true, "sessionId": "session-1", "workspaceRoot": "/workspace", "messageCount": 7}
			case "autohand.browserHandoff.attachLatest":
				responseResult = map[string]interface{}{"success": false}
			case "autohand.automode.start":
				responseResult = map[string]interface{}{"success": true, "sessionId": "auto-1"}
			case "autohand.automode.status":
				responseResult = map[string]interface{}{
					"active": true, "paused": false,
					"state": map[string]interface{}{
						"sessionId": "auto-1", "status": "running", "currentIteration": 4, "maxIterations": 12,
						"filesCreated": 2, "filesModified": 5, "branch": "autohand/auto-1",
						"lastCheckpoint": map[string]interface{}{"commit": "abc123", "message": "checkpoint", "timestamp": "2026-07-20T01:02:00Z"},
					},
				}
			case "autohand.automode.pause":
				responseResult = map[string]interface{}{"success": false, "error": "No auto-mode session is running"}
			case "autohand.autoresearch.start":
				result["instruction"] = "Run the next experiment"
			case "autohand.autoresearch.status":
				result["active"] = true
				result["statusText"] = "Auto-research active"
				result["runsLogged"] = 2
			case "autohand.autoresearch.history":
				result["attempts"] = []interface{}{}
			case "autohand.autoresearch.rescore":
				result["decisions"] = []interface{}{}
			case "autohand.autoresearch.pareto":
				result["attemptIds"] = []interface{}{}
			case "autohand.autoresearch.pin":
				result["attemptId"] = "attempt-1"
				result["pinned"] = true
			case "autohand.autoresearch.prune":
				result["applied"] = false
				result["candidates"] = []interface{}{}
				result["bytesFreed"] = 0
				result["remainingBytes"] = 0
			case "autohand.goal.listTemplates":
				responseResult = []interface{}{}
			case "autohand.getSkillsRegistry":
				responseResult = map[string]interface{}{
					"success": true,
					"skills": []interface{}{map[string]interface{}{
						"id": "skill-1", "name": "review", "description": "Review code", "category": "quality",
					}},
					"categories": []interface{}{map[string]interface{}{"name": "quality", "count": 1}},
				}
			case "autohand.installSkill":
				responseResult = map[string]interface{}{"success": true, "skillName": "review", "path": ".autohand/skills/review"}
			case "autohand.mcp.listServers":
				responseResult = map[string]interface{}{"servers": []interface{}{map[string]interface{}{"name": "github", "status": "connected", "toolCount": 2}}}
			case "autohand.mcp.listTools":
				responseResult = map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": "issues", "description": "List issues", "serverName": "github"}}}
			case "autohand.mcp.getServerConfigs":
				responseResult = map[string]interface{}{"configs": []interface{}{map[string]interface{}{"name": "github", "transport": "stdio", "command": "gh-mcp", "args": []string{"serve"}, "autoConnect": true}}}
			}
			response, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result":  responseResult,
			})
			transport.handleLine(string(response))
		}
	}()

	return client, requests, func() {
		_ = writer.Close()
		_ = reader.Close()
	}
}

func boolPointer(value bool) *bool { return &value }
