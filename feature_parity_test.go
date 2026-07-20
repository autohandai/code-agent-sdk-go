package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFormatSlashCommand(t *testing.T) {
	got, err := FormatSlashCommand(" /deep-research ", "  find ", "", "evidence ")
	if err != nil || got != "/deep-research find evidence" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := FormatSlashCommand("deep-research"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGoalRPCMethodsAndSnakeCaseParams(t *testing.T) {
	client, requests, closeTransport := newRPCTestClient(t)
	defer closeTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	budget := 100
	if _, err := client.GetGoal(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateGoal(ctx, &GoalCreateParams{Objective: "ship", TokenBudget: &budget}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateGoal(ctx, &GoalUpdateParams{TokenBudget: new(*int)}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.QueueGoal(ctx, &GoalCreateParams{Objective: "next"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.StartQueuedGoal(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListGoalTemplates(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ClearGoal(ctx); err != nil {
		t.Fatal(err)
	}
	want := []string{"autohand.goal.get", "autohand.goal.create", "autohand.goal.update", "autohand.goal.queue", "autohand.goal.startQueued", "autohand.goal.listTemplates", "autohand.goal.clear"}
	got := make([]capturedRPCRequest, len(want))
	for i := range got {
		got[i] = <-requests
	}
	methods := make([]string, len(got))
	for i := range got {
		methods[i] = got[i].Method
	}
	if !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods %v", methods)
	}
	var create map[string]interface{}
	_ = json.Unmarshal(got[1].Params, &create)
	if create["token_budget"] != float64(100) || create["tokenBudget"] != nil {
		t.Fatalf("params %v", create)
	}
	if string(got[2].Params) != `{"token_budget":null}` {
		t.Fatalf("update params %s", got[2].Params)
	}
}

func TestBuildCLIArgsCurrentRuntimeContract(t *testing.T) {
	no := false
	got := buildCLIArgs(&Config{Bare: true, IdleLogout: &no, Fork: true, NoAgentsMd: true, NoContextCompact: true, SkillSources: []string{"user", "project"}, InstallMissingSkills: true, DisplayLanguage: "pt-BR", SystemPromptFile: "system.md", AppendSystemPromptFile: "append.md", MCPConfig: "mcp.json", Agents: "agents.json", PluginDir: "plugins"})
	wantParts := []string{"--bare", "--no-idle-logout", "--fork", "--no-agents-md", "--no-context-compact", "--skill-sources", "user,project", "--install-missing-skills", "--display-language", "pt-BR", "--system-prompt-file", "system.md", "--append-system-prompt-file", "append.md", "--mcp-config", "mcp.json", "--agents", "agents.json", "--plugin-dir", "plugins"}
	joined := strings.Join(got, " ")
	for _, part := range wantParts {
		if !strings.Contains(joined, part) {
			t.Errorf("args missing %q: %v", part, got)
		}
	}
}

func TestAutohandAIEnvironmentAndTurnUsage(t *testing.T) {
	env := buildCLIEnv(&Config{Provider: ProviderAutohandAI, APIKey: "key", BaseURL: "https://api", Env: map[string]string{"AUTOHAND_AI_PLAN": "max"}}, nil)
	joined := strings.Join(env, "\n")
	for _, value := range []string{"AUTOHAND_AI_PLAN=cloud", "AUTOHAND_AI_API_KEY=key", "AUTOHAND_AI_BASE_URL=https://api", "AUTOHAND_AI_PLAN=max"} {
		if !strings.Contains(joined, value) {
			t.Errorf("missing %s", value)
		}
	}
	var event TurnEndEvent
	if err := json.Unmarshal([]byte(`{"tokensUsed":9,"tokensUsageStatus":"actual","durationMs":4,"contextPercent":2.5}`), &event); err != nil {
		t.Fatal(err)
	}
	if event.TokensUsageStatus != "actual" || event.TokensUsed != 9 {
		t.Fatalf("event %+v", event)
	}
}

func TestStartupAppliesFeatureSettingsRPC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake CLI")
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "autohand")
	logPath := filepath.Join(dir, "requests.log")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--test-log" ]; then log="$2"; shift; fi
  shift
done
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$log"
  id=$(printf '%s\n' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  printf '{"jsonrpc":"2.0","id":%s,"result":{}}\n' "$id"
done
`
	if err := os.WriteFile(cli, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	yes := true
	sdk := NewSDK(&Config{CLIPath: cli, Features: &FeatureFlagSettings{SlashGoal: &yes}, ExtraArgs: []string{"--test-log", logPath}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sdk.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := sdk.Stop(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := `"method":"autohand.applyFlagSettings"`
	if !strings.Contains(string(contents), want) || !strings.Contains(string(contents), `"features":{"slashGoal":true}`) {
		t.Fatalf("startup request missing feature settings: %s", fmt.Sprintf("%s", contents))
	}
}
