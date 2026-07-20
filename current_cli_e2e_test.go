//go:build !windows

package autohand

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type currentCLIFixture struct {
	sdk     *SDK
	ctx     context.Context
	logPath string
}

func newCurrentCLIFixture(t *testing.T, result, notification string) *currentCLIFixture {
	t.Helper()
	dir := t.TempDir()
	cli := filepath.Join(dir, "autohand")
	logPath := filepath.Join(dir, "requests.log")
	script := `#!/bin/sh
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$AUTOHAND_TEST_LOG"
  id=$(printf '%s\n' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$line" in
    *autohand.getState*) response='{}' ;;
    *autohand.prompt*)
      if [ -n "$AUTOHAND_TEST_NOTIFICATION" ]; then
        printf '%s\n' "$AUTOHAND_TEST_NOTIFICATION"
      fi
      response='{"success":true}'
      ;;
    *) response="$AUTOHAND_TEST_RESULT" ;;
  esac
  printf '{"jsonrpc":"2.0","id":%s,"result":%s}\n' "$id" "$response"
done
`
	if err := os.WriteFile(cli, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	sdk := NewSDK(&Config{
		CLIPath: cli,
		Timeout: 2_000,
		Env: map[string]string{
			"AUTOHAND_TEST_LOG":          logPath,
			"AUTOHAND_TEST_RESULT":       result,
			"AUTOHAND_TEST_NOTIFICATION": notification,
		},
	})
	if err := sdk.Start(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sdk.Stop()
		cancel()
	})
	return &currentCLIFixture{sdk: sdk, ctx: ctx, logPath: logPath}
}

func (f *currentCLIFixture) assertRequest(t *testing.T, method string, params ...string) {
	t.Helper()
	contents, err := os.ReadFile(f.logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(contents)
	if !strings.Contains(log, `"method":"`+method+`"`) {
		t.Fatalf("request log does not contain %s: %s", method, log)
	}
	for _, param := range params {
		if !strings.Contains(log, param) {
			t.Fatalf("request log does not contain %s: %s", param, log)
		}
	}
}

func TestAcknowledgePermissionE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true}`, "")
	result, err := fixture.sdk.AcknowledgePermission(fixture.ctx, "permission-1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.permissionAcknowledged", `"requestId":"permission-1"`)

	if _, err := fixture.sdk.AcknowledgePermission(fixture.ctx, "  "); err == nil {
		t.Fatal("expected blank request ID to fail before transport")
	}
}

func TestRespondToDirectoryAccessE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true}`, "")
	result, err := fixture.sdk.RespondToDirectoryAccess(fixture.ctx, "directory-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.directoryAccessResponse", `"requestId":"directory-1"`, `"granted":false`)

	if _, err := fixture.sdk.RespondToDirectoryAccess(fixture.ctx, "", true); err == nil {
		t.Fatal("expected blank request ID to fail before transport")
	}
}

func TestAcknowledgeDirectoryAccessE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true}`, "")
	result, err := fixture.sdk.AcknowledgeDirectoryAccess(fixture.ctx, "directory-2")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.directoryAccessAcknowledged", `"requestId":"directory-2"`)

	if _, err := fixture.sdk.AcknowledgeDirectoryAccess(fixture.ctx, "\t"); err == nil {
		t.Fatal("expected blank request ID to fail before transport")
	}
}

func TestDecideChangesE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"appliedCount":1,"skippedCount":2,"errors":[]}`, "")
	result, err := fixture.sdk.DecideChanges(fixture.ctx, &ChangesDecisionParams{
		BatchID:           "batch-1",
		Action:            ChangesAcceptSelected,
		SelectedChangeIDs: []string{"change-2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.AppliedCount != 1 || result.SkippedCount != 2 {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.changesDecision", `"batchId":"batch-1"`, `"action":"accept_selected"`, `"selectedChangeIds":["change-2"]`)

	if _, err := fixture.sdk.DecideChanges(fixture.ctx, &ChangesDecisionParams{BatchID: "batch-1", Action: ChangesAcceptSelected}); err == nil {
		t.Fatal("expected empty selected change list to fail before transport")
	}
}

func TestGetHistoryE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"sessions":[{"sessionId":"session-1","createdAt":"now","lastActiveAt":"later","projectName":"tin","model":"gpt-5","messageCount":4,"status":"completed"}],"currentPage":2,"totalPages":3,"totalItems":5}`, "")
	result, err := fixture.sdk.GetHistory(fixture.ctx, &GetHistoryParams{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalItems != 5 || len(result.Sessions) != 1 || result.Sessions[0].Status != SessionHistoryCompleted {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.getHistory", `"page":2`, `"pageSize":1`)

	if _, err := fixture.sdk.GetHistory(fixture.ctx, &GetHistoryParams{Page: -1}); err == nil {
		t.Fatal("expected negative page to fail before transport")
	}
}

func TestGetSessionE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"sessionId":"session-1","projectName":"tin","model":"gpt-5","messageCount":1,"status":"completed","createdAt":"now","lastActiveAt":"later","messages":[{"id":"message-1","role":"assistant","content":"done","timestamp":"later"}],"workspaceRoot":"/workspace"}`, "")
	result, err := fixture.sdk.GetSession(fixture.ctx, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	details, ok := result.(SessionDetails)
	if !ok || !details.Succeeded() || len(details.Messages) != 1 {
		t.Fatalf("result = %#v", result)
	}
	fixture.assertRequest(t, "autohand.getSession", `"sessionId":"session-1"`)

	failureFixture := newCurrentCLIFixture(t, `{"success":false,"error":"not found"}`, "")
	failure, err := failureFixture.sdk.GetSession(failureFixture.ctx, "missing")
	if err != nil {
		t.Fatal(err)
	}
	missing, ok := failure.(SessionLookupFailure)
	if !ok || missing.Succeeded() || missing.Error != "not found" {
		t.Fatalf("failure = %#v", failure)
	}

	malformed := newCurrentCLIFixture(t, `{"success":true,"sessionId":"partial"}`, "")
	if _, err := malformed.sdk.GetSession(malformed.ctx, "partial"); err == nil {
		t.Fatal("expected malformed success payload to fail")
	}
}

func TestAttachSessionE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"sessionId":"session-2","workspaceRoot":"/workspace","messageCount":6}`, "")
	result, err := fixture.sdk.AttachSession(fixture.ctx, "session-2")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.SessionID != "session-2" || result.MessageCount != 6 {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.session.attach", `"sessionId":"session-2"`)

	if _, err := fixture.sdk.AttachSession(fixture.ctx, " "); err == nil {
		t.Fatal("expected blank session ID to fail before transport")
	}
}

func TestSetYoloE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"expiresIn":45}`, "")
	params := &YoloSetParams{Pattern: "*", TimeoutSeconds: 45}
	canonical, err := fixture.sdk.SetYolo(fixture.ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := fixture.sdk.SetYoloAlias(fixture.ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.ExpiresIn == nil || *canonical.ExpiresIn != 45 || alias.ExpiresIn == nil || *alias.ExpiresIn != 45 {
		t.Fatalf("canonical = %+v, alias = %+v", canonical, alias)
	}
	fixture.assertRequest(t, "autohand.yoloSet", `"pattern":"*"`, `"timeoutSeconds":45`)
	fixture.assertRequest(t, "autohand.yolo.set")

	if _, err := fixture.sdk.SetYolo(fixture.ctx, &YoloSetParams{TimeoutSeconds: -1}); err == nil {
		t.Fatal("expected negative timeout to fail before transport")
	}
}

func TestSetVSCodeMCPToolsE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true}`, "")
	result, err := fixture.sdk.SetVSCodeMCPTools(fixture.ctx, &MCPSetVSCodeToolsParams{Tools: []MCPVSCodeTool{{
		Name: "issues", Description: "List issues", ServerName: "github",
		InputSchema: &MCPInputSchema{Type: "object", Properties: map[string]interface{}{"state": map[string]interface{}{"type": "string"}}, Required: []string{"state"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.mcp.setVscodeTools", `"serverName":"github"`, `"inputSchema":{"type":"object"`, `"required":["state"]`)

	if _, err := fixture.sdk.SetVSCodeMCPTools(fixture.ctx, &MCPSetVSCodeToolsParams{Tools: []MCPVSCodeTool{{Name: "broken"}}}); err == nil {
		t.Fatal("expected malformed tool descriptor to fail before transport")
	}
}

func TestRespondToMCPInvocationE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true}`, "")
	result, err := fixture.sdk.RespondToMCPInvocation(fixture.ctx, &MCPInvocationResponseParams{
		RequestID: "invoke-1", Success: false, Error: "tool unavailable",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.mcp.invokeResponse", `"requestId":"invoke-1"`, `"success":false`, `"error":"tool unavailable"`)

	invalidResult := "unexpected"
	if _, err := fixture.sdk.RespondToMCPInvocation(fixture.ctx, &MCPInvocationResponseParams{RequestID: "invoke-2", Result: &invalidResult, Error: "failed"}); err == nil {
		t.Fatal("expected ambiguous failed response to fail before transport")
	}
}

func TestRecommendProjectLearningE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"projectSummary":"Go SDK","audit":[{"skill":"old","status":"outdated","reason":"stale"}],"recommendations":[{"slug":"go-testing","score":0.95,"reason":"missing tests"}],"gapAnalysis":"Add integration coverage"}`, "")
	result, err := fixture.sdk.RecommendProjectLearning(fixture.ctx, &LearnRecommendParams{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || len(result.Audit) != 1 || result.Audit[0].Status != LearningAuditOutdated || result.GapAnalysis == nil {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.learn.recommend", `"deep":true`)
}

func TestUpdateProjectLearningE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"updated":1,"unchanged":2,"results":[{"name":"testing","status":"updated"}]}`, "")
	result, err := fixture.sdk.UpdateProjectLearning(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Updated != 1 || result.Results[0].Status != LearningUpdated {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.learn.update", `"params":{}`)
}

func TestGenerateProjectSkillE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true,"skillName":"release","skillPath":".autohand/skills/release"}`, "")
	result, err := fixture.sdk.GenerateProjectSkill(fixture.ctx, &LearnGenerateParams{Scope: SkillGenerationProject})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.SkillName != "release" {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.learn.generate", `"scope":"project"`)

	if _, err := fixture.sdk.GenerateProjectSkill(fixture.ctx, &LearnGenerateParams{}); err == nil {
		t.Fatal("expected invalid scope to fail before transport")
	}
}

func TestGetToolsRegistryE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"tools":[{"name":"read_file","description":"Read a file","requiresApproval":false,"source":"builtin"},{"name":"review","description":"Review code","source":"extension","scope":"project","extensionId":"quality"}],"diagnostics":[{"file":"broken.json","reason":"invalid schema"}]}`, "")
	result, err := fixture.sdk.GetToolsRegistry(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 2 || result.Tools[1].Source != ToolRegistryExtension || result.Tools[0].RequiresApproval == nil || *result.Tools[0].RequiresApproval || len(result.Diagnostics) != 1 {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.getToolsRegistry", `"params":{}`)
}

func TestSetContextCompactE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"enabled":false}`, "")
	result, err := fixture.sdk.SetContextCompact(fixture.ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Enabled || !fixture.sdk.Config().NoContextCompact {
		t.Fatalf("result = %+v, config = %+v", result, fixture.sdk.Config())
	}
	fixture.assertRequest(t, "autohand.setContextCompact", `"enabled":false`)
}

func TestAutomodeIterationEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.automode.iteration","params":{"sessionId":"auto-1","iteration":3,"actions":["edit","test"],"tokensUsed":321,"timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(AutomodeIterationEvent)
		if !ok || event.Iteration != 3 || event.TokensUsed == nil || *event.TokensUsed != 321 {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed auto-mode iteration event")
	}
	fixture.assertRequest(t, "autohand.prompt")
}

func TestUnknownNotificationFallbackE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.future.event","params":{"value":7}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(GenericEvent)
		if !ok || event.Method != "autohand.future.event" || !strings.Contains(string(event.Params), `"value":7`) {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for generic fallback event")
	}
}

func TestAutomodeCompleteEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.automode.complete","params":{"sessionId":"auto-1","iterations":4,"filesCreated":2,"filesModified":5,"timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(AutomodeCompleteEvent)
		if !ok || event.Iterations != 4 || event.FilesModified != 5 {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed auto-mode completion event")
	}
}

func TestAutomodeErrorEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.automode.error","params":{"sessionId":"auto-1","error":"iteration failed","timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(AutomodeErrorEvent)
		if !ok || event.Error != "iteration failed" {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed auto-mode error event")
	}
}

func TestHookPreToolEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.hook.preTool","params":{"toolId":"tool-1","toolName":"read_file","args":{"path":"README.md"},"timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(HookPreToolEvent)
		if !ok || event.ToolName != "read_file" || event.Args["path"] != "README.md" {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed pre-tool hook event")
	}
}

func TestHookPostToolEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.hook.postTool","params":{"toolId":"tool-1","toolName":"read_file","success":true,"duration":12.5,"output":"contents","timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(HookPostToolEvent)
		if !ok || !event.Success || event.Duration != 12.5 || event.Output == nil || *event.Output != "contents" {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed post-tool hook event")
	}
}

func TestHookPrePromptEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.hook.prePrompt","params":{"instruction":"Review the SDK","mentionedFiles":["sdk.go","rpc.go"],"timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(HookPrePromptEvent)
		if !ok || event.Instruction != "Review the SDK" || len(event.MentionedFiles) != 2 {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed pre-prompt hook event")
	}
}

func TestHookPostResponseEventE2E(t *testing.T) {
	notification := `{"jsonrpc":"2.0","method":"autohand.hook.postResponse","params":{"tokensUsed":0,"tokensUsageStatus":"unavailable","toolCallsCount":2,"duration":18.75,"timestamp":"now"}}`
	fixture := newCurrentCLIFixture(t, `{"success":true}`, notification)
	events, err := fixture.sdk.Events(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.sdk.Prompt(fixture.ctx, &PromptParams{Message: "emit"}); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-events:
		event, ok := received.(HookPostResponseEvent)
		if !ok || event.ToolCallsCount != 2 || event.TokensUsageStatus == nil || *event.TokensUsageStatus != TokenAccountingUnavailable {
			t.Fatalf("event = %#v", received)
		}
	case <-fixture.ctx.Done():
		t.Fatal("timed out waiting for typed post-response hook event")
	}
}
