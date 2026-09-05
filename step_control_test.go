package autohand

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPromptWireContract(t *testing.T) {
	client, requests, closeClient := newRPCTestClient(t)
	defer closeClient()
	err := client.Prompt(context.Background(), &PromptParams{
		Message: "inspect", Context: &PromptContext{Files: []string{"main.go"}},
		ThinkingLevel: "extended",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := <-requests
	var params map[string]json.RawMessage
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatal(err)
	}
	if string(params["message"]) != `"inspect"` || string(params["context"]) != `{"files":["main.go"]}` || string(params["thinkingLevel"]) != `"extended"` {
		t.Fatalf("invalid CLI prompt: %s", request.Params)
	}
}

func newStepControlAgent(t *testing.T) (*Agent, context.Context, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX subprocess fixture")
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "autohand")
	log := filepath.Join(dir, "requests.jsonl")
	script := `#!/bin/sh
step() {
  if [ "$AUTOHAND_TEST_MALFORMED_STEP" = 1 ]; then
    printf '{"jsonrpc":"2.0","method":"autohand.stepEnd","params":{"stepId":"bad"}}\n'
    return
  fi
  printf '{"jsonrpc":"2.0","method":"autohand.stepEnd","params":{"stepId":"step-%s","step":{"stepNumber":%s,"toolCalls":[{"id":"tool-1","tool":"read_file","args":{"path":"main.go"}}],"toolResults":[{"tool":"read_file","success":true,"output":"saved evidence"}]},"timestamp":"now"}}\n' "$count" "$count"
}
finish() {
  printf '{"jsonrpc":"2.0","method":"autohand.turnEnd","params":{"turnId":"turn-1","reason":"%s","timestamp":"now"}}\n' "$1"
  printf '{"jsonrpc":"2.0","id":%s,"result":{"success":true}}\n' "$prompt"
  prompt=''
}
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$AUTOHAND_TEST_LOG"
  id=$(printf '%s\n' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$line" in
    *autohand.getState*) printf '{"jsonrpc":"2.0","id":%s,"result":{}}\n' "$id" ;;
    *autohand.prompt*)
      case "$line" in
        *'"message":"exit"'*) printf '{"jsonrpc":"2.0","id":%s,"result":{"success":true}}\n' "$id"; exit 0 ;;
      esac
      if [ -n "$prompt" ]; then
        printf '{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"overlapping prompt"}}\n' "$id"
      else
        prompt="$id"
        case "$line" in
          *'"stopWhen":{"mode":"host"}'*) count=1; step ;;
          *) printf '{"jsonrpc":"2.0","method":"autohand.messageEnd","params":{"content":"continued","timestamp":"now"}}\n'; finish completed ;;
        esac
      fi ;;
    *autohand.stepDecision*)
      if [ "$AUTOHAND_TEST_REJECT_STEP" = 1 ]; then
        printf '{"jsonrpc":"2.0","id":%s,"result":{"success":false}}\n' "$id"
        continue
      fi
      printf '{"jsonrpc":"2.0","id":%s,"result":{"success":true}}\n' "$id"
      case "$line" in
        *'"stop":true'*) finish stop_condition ;;
        *) count=$((count + 1)); step ;;
      esac ;;
    *autohand.abort*)
      printf '{"jsonrpc":"2.0","id":%s,"result":{"success":true}}\n' "$id"
      if [ -n "$prompt" ]; then finish aborted; fi ;;
  esac
done
`
	if err := os.WriteFile(cli, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	agent, err := NewAgent(ctx, &Config{CLIPath: cli, Timeout: 2000, Env: map[string]string{"AUTOHAND_TEST_LOG": log}})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = agent.Close(); cancel() })
	return agent, ctx, log
}

func TestStopWhenPausesAndContinuesSameSession(t *testing.T) {
	agent, ctx, log := newStepControlAgent(t)
	condition, err := IsStepCount(2)
	if err != nil {
		t.Fatal(err)
	}
	result, err := agent.Run(ctx, "inspect", &PromptParams{StopWhen: []StopCondition{condition}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "stopped" || len(result.Steps) != 2 || result.Steps[1].ToolResults[0].Output != "saved evidence" {
		t.Fatalf("result: %+v", result)
	}
	continued, err := agent.Run(ctx, "continue", nil)
	if err != nil || continued.Status != "completed" || continued.Text != "continued" {
		t.Fatalf("continue: %+v, %v", continued, err)
	}
	contents, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), `"stop":false`) || !strings.Contains(string(contents), `"stop":true`) {
		t.Fatalf("decisions: %s", contents)
	}
}

func TestStopConditionFailureSettlesBeforeReturningError(t *testing.T) {
	agent, ctx, _ := newStepControlAgent(t)
	failure := errors.New("predicate failed")
	_, err := agent.Run(ctx, "inspect", &PromptParams{StopWhen: []StopCondition{func(context.Context, []AgentStep) (bool, error) { return false, failure }}})
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v", err)
	}
	result, err := agent.Run(ctx, "continue", nil)
	if err != nil || result.Status != "completed" {
		t.Fatalf("continue: %+v, %v", result, err)
	}
}

func TestStopHelpersValidateInputs(t *testing.T) {
	if _, err := IsStepCount(0); err == nil {
		t.Fatal("accepted zero steps")
	}
	if _, err := HasToolCall(" "); err == nil {
		t.Fatal("accepted empty tool name")
	}
	condition, err := HasToolCall("read_file")
	if err != nil {
		t.Fatal(err)
	}
	stop, err := condition(context.Background(), []AgentStep{{ToolCalls: []AgentStepToolCall{{Tool: "read_file"}}}})
	if err != nil || !stop {
		t.Fatalf("stop=%v err=%v", stop, err)
	}
}

func TestRunWaitIsRepeatable(t *testing.T) {
	agent, ctx, _ := newStepControlAgent(t)
	run, err := agent.Send(ctx, "continue", nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := run.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := run.Wait(ctx)
	if err != nil || first.Text != second.Text {
		t.Fatalf("second wait: %+v, %v", second, err)
	}
}

func TestQueuedRunAbortDoesNotAbortActiveRun(t *testing.T) {
	agent, ctx, _ := newStepControlAgent(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	first, err := agent.Send(ctx, "inspect", &PromptParams{StopWhen: []StopCondition{func(ctx context.Context, _ []AgentStep) (bool, error) {
		close(entered)
		select {
		case <-release:
			return true, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}}})
	if err != nil {
		t.Fatal(err)
	}
	firstDone := make(chan error, 1)
	go func() {
		result, err := first.Wait(ctx)
		if err == nil && result.Status != "stopped" {
			err = errors.New("active run was aborted")
		}
		firstDone <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	second, err := agent.Send(ctx, "queued", nil)
	if err != nil {
		t.Fatal(err)
	}
	secondDone := make(chan error, 1)
	go func() { _, err := second.Wait(ctx); secondDone <- err }()
	if err := second.Abort(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("queued run: %v", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}

func TestAbortWhileStopConditionWaitsDrainsTheTurn(t *testing.T) {
	agent, ctx, log := newStepControlAgent(t)
	entered := make(chan struct{})
	conditionDone := make(chan struct{})
	run, err := agent.Send(ctx, "inspect", &PromptParams{StopWhen: []StopCondition{func(ctx context.Context, _ []AgentStep) (bool, error) {
		close(entered)
		<-ctx.Done()
		close(conditionDone)
		return true, nil
	}}})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := run.Wait(ctx); done <- err }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := run.Abort(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("abort: %v", err)
	}
	select {
	case <-conditionDone:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	result, err := agent.Run(ctx, "continue", nil)
	if err != nil || result.Text != "continued" {
		t.Fatalf("continue: %+v, %v", result, err)
	}
	contents, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "autohand.stepDecision") {
		t.Fatalf("sent a decision after abort: %s", contents)
	}
}

func TestPromptEvaluatesStopConditions(t *testing.T) {
	agent, ctx, _ := newStepControlAgent(t)
	condition, err := IsStepCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := agent.sdk.Prompt(ctx, &PromptParams{Message: "inspect", StopWhen: []StopCondition{condition}}); err != nil {
		t.Fatal(err)
	}
}

func TestStepNotificationValidation(t *testing.T) {
	for _, payload := range []string{
		`{}`,
		`{"stepId":"s","timestamp":"t","step":{"stepNumber":1,"toolCalls":[],"toolResults":[{"tool":"read_file"}]}}`,
		`{"stepId":"s","timestamp":"t","step":{"stepNumber":1,"toolCalls":[{"tool":"read_file","args":null}],"toolResults":[]}}`,
	} {
		if _, err := parseStepEnd(json.RawMessage(payload)); err == nil {
			t.Fatalf("accepted malformed step: %s", payload)
		}
	}
}

func TestStepProtocolFailuresAbortAndAllowContinuation(t *testing.T) {
	for _, scenario := range []struct{ env, message string }{
		{"AUTOHAND_TEST_MALFORMED_STEP", "malformed autohand.stepEnd"},
		{"AUTOHAND_TEST_REJECT_STEP", "rejected step decision"},
	} {
		t.Run(scenario.env, func(t *testing.T) {
			t.Setenv(scenario.env, "1")
			agent, ctx, _ := newStepControlAgent(t)
			condition, err := IsStepCount(1)
			if err != nil {
				t.Fatal(err)
			}
			_, err = agent.Run(ctx, "inspect", &PromptParams{StopWhen: []StopCondition{condition}})
			if err == nil || !strings.Contains(err.Error(), scenario.message) {
				t.Fatalf("protocol error: %v", err)
			}
			result, err := agent.Run(ctx, "continue", nil)
			if err != nil || result.Text != "continued" {
				t.Fatalf("continue: %+v, %v", result, err)
			}
		})
	}
}

func TestEventOverflowFailsExplicitly(t *testing.T) {
	client := NewRPCClient(&Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := client.Events(ctx)
	for i := 0; i < 300; i++ {
		client.queueEvent(MessageUpdateEvent{Delta: "token"})
	}
	found := false
	for {
		select {
		case event, ok := <-events:
			if !ok {
				if !found {
					t.Fatal("overflow lost without an error")
				}
				return
			}
			if failure, ok := event.(ErrorEvent); ok && errors.Is(failure.Err, ErrEventOverflow) {
				found = true
			}
		case <-time.After(time.Second):
			t.Fatal("overflowed stream remained open")
		}
	}
}

func TestCLIExitAfterPromptAcceptanceFailsTheRun(t *testing.T) {
	agent, ctx, _ := newStepControlAgent(t)
	_, err := agent.Run(ctx, "exit", nil)
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("CLI exit: %v", err)
	}
}
