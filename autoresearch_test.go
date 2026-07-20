package autohand

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestAutoresearchRPCMethods(t *testing.T) {
	client, requests, closeTransport := newRPCTestClient(t)
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

func boolPointer(value bool) *bool { return &value }
