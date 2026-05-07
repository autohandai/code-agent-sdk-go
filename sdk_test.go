package autohand

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDetectProviderFromModel(t *testing.T) {
	tests := []struct {
		model    string
		expected ProviderName
	}{
		{"", ProviderOpenRouter},
		{"gpt-4", ProviderOpenAI},
		{"claude-sonnet", ProviderOpenRouter},
		{"deepseek-chat", ProviderDeepSeek},
		{"grok-2", ProviderXai},
		{"openrouter/auto", ProviderOpenRouter},
	}

	for _, tt := range tests {
		result := DetectProviderFromModel(tt.model)
		if result != tt.expected {
			t.Errorf("DetectProviderFromModel(%q) = %q, want %q", tt.model, result, tt.expected)
		}
	}
}

func TestAllowDecision(t *testing.T) {
	tests := []struct {
		scope    DecisionScope
		expected PermissionDecision
	}{
		{ScopeOnce, DecisionAllowOnce},
		{ScopeSession, DecisionAllowSession},
		{ScopeProject, DecisionAllowAlwaysProject},
		{ScopeUser, DecisionAllowAlwaysUser},
	}

	for _, tt := range tests {
		result := allowDecision(tt.scope)
		if result != tt.expected {
			t.Errorf("allowDecision(%q) = %q, want %q", tt.scope, result, tt.expected)
		}
	}
}

func TestDenyDecision(t *testing.T) {
	tests := []struct {
		scope    DecisionScope
		expected PermissionDecision
	}{
		{ScopeOnce, DecisionDenyOnce},
		{ScopeSession, DecisionDenySession},
		{ScopeProject, DecisionDenyAlwaysProject},
		{ScopeUser, DecisionDenyAlwaysUser},
	}

	for _, tt := range tests {
		result := denyDecision(tt.scope)
		if result != tt.expected {
			t.Errorf("denyDecision(%q) = %q, want %q", tt.scope, result, tt.expected)
		}
	}
}

func TestNewSDK(t *testing.T) {
	sdk := NewSDK(nil)
	if sdk == nil {
		t.Fatal("NewSDK returned nil")
	}
	if sdk.IsStarted() {
		t.Error("SDK should not be started initially")
	}
}

func TestSDKConfig(t *testing.T) {
	cfg := &Config{
		Model: "gpt-4",
		Debug: true,
	}
	sdk := NewSDK(cfg)
	got := sdk.Config()
	if got.Model != "gpt-4" {
		t.Errorf("Config().Model = %q, want %q", got.Model, "gpt-4")
	}
	if !got.Debug {
		t.Error("Config().Debug should be true")
	}
}

func TestSDKUpdateConfig(t *testing.T) {
	sdk := NewSDK(nil)
	sdk.UpdateConfig(&Config{
		Model: "claude-sonnet",
		Debug: true,
	})
	cfg := sdk.Config()
	if cfg.Model != "claude-sonnet" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-sonnet")
	}
}

func TestNewAgent(t *testing.T) {
	cfg := &Config{
		Model: "gpt-4",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	agent, err := NewAgent(ctx, cfg)
	if err != nil {
		t.Logf("Could not start agent (expected if CLI not installed): %v", err)
		return
	}
	defer agent.Close()

	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}
}

func TestToPromptParams(t *testing.T) {
	params := toPromptParams("hello", nil)
	if params.Message != "hello" {
		t.Errorf("Message = %q, want %q", params.Message, "hello")
	}

	existing := &PromptParams{Message: "existing"}
	params = toPromptParams(existing, nil)
	if params.Message != "existing" {
		t.Errorf("Message = %q, want %q", params.Message, "existing")
	}

	opts := &PromptParams{
		ThinkingLevel: "extended",
	}
	params = toPromptParams("hello", opts)
	if params.Message != "hello" {
		t.Errorf("Message = %q, want %q", params.Message, "hello")
	}
	if params.ThinkingLevel != "extended" {
		t.Errorf("ThinkingLevel = %q, want %q", params.ThinkingLevel, "extended")
	}
}

func TestEventTypes(t *testing.T) {
	events := []Event{
		AgentStartEvent{Type: "agent_start", SessionID: "s1", Model: "m1", Workspace: "w1", Timestamp: "t1"},
		AgentEndEvent{Type: "agent_end", SessionID: "s1", Reason: "completed", Timestamp: "t1"},
		TurnStartEvent{Type: "turn_start", TurnID: "t1", Timestamp: "ts1"},
		TurnEndEvent{Type: "turn_end", TurnID: "t1", Timestamp: "ts1"},
		MessageStartEvent{Type: "message_start", MessageID: "m1", Role: "assistant", Timestamp: "ts1"},
		MessageUpdateEvent{Type: "message_update", Delta: "hello", Timestamp: "ts1"},
		MessageEndEvent{Type: "message_end", MessageID: "m1", Content: "hello world", Timestamp: "ts1"},
		ToolStartEvent{Type: "tool_start", ToolID: "t1", ToolName: "read_file", Timestamp: "ts1"},
		ToolUpdateEvent{Type: "tool_update", ToolID: "t1", Output: "output", Stream: "stdout", Timestamp: "ts1"},
		ToolEndEvent{Type: "tool_end", ToolID: "t1", ToolName: "read_file", Success: true, Timestamp: "ts1"},
		FileModifiedEvent{Type: "file_modified", FilePath: "/f", ChangeType: "modify", ToolID: "t1", Timestamp: "ts1"},
		PermissionRequestEvent{Type: "permission_request", RequestID: "r1", Tool: "run_command", Description: "desc", Timestamp: "ts1"},
		ErrorEvent{Type: "error", Code: 1, Message: "err", Recoverable: true, Timestamp: "ts1"},
	}

	for i, e := range events {
		if e.eventType() == "" {
			t.Errorf("event[%d] has empty type", i)
		}
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		strs     []string
		sep      string
		expected string
	}{
		{nil, ",", ""},
		{[]string{}, ",", ""},
		{[]string{"a"}, ",", "a"},
		{[]string{"a", "b"}, ",", "a,b"},
		{[]string{"a", "b", "c"}, "-", "a-b-c"},
	}

	for _, tt := range tests {
		result := strings.Join(tt.strs, tt.sep)
		if result != tt.expected {
			t.Errorf("strings.Join(%v, %q) = %q, want %q", tt.strs, tt.sep, result, tt.expected)
		}
	}
}

func TestLineReader(t *testing.T) {
	t.Skip("LineReader requires a real io.Reader")
}

func TestTransportDetectBinary(t *testing.T) {
	transport := NewTransport(&Config{})
	path, err := transport.detectCLIBinary()
	if err != nil {
		t.Skipf("CLI binary not found (expected if not installed): %v", err)
	}
	if path == "" {
		t.Error("detectCLIBinary returned empty path")
	}
	t.Logf("Detected binary: %s", path)
}

func TestParseJsonText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkKey string
	}{
		{
			name:     "direct object",
			input:    `{"ok": true}`,
			wantErr:  false,
			checkKey: "ok",
		},
		{
			name:     "fenced json",
			input:    "```json\n{\"ok\": true}\n```",
			wantErr:  false,
			checkKey: "ok",
		},
		{
			name:     "embedded json",
			input:    "Some text before {\"ok\": true} and after",
			wantErr:  false,
			checkKey: "ok",
		},
		{
			name:    "no json",
			input:   "just plain text",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJsonText(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %v", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkKey != "" {
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", result)
				}
				if _, exists := m[tt.checkKey]; !exists {
					t.Errorf("expected key %q in result", tt.checkKey)
				}
			}
		})
	}
}

func TestStructuredOutputError(t *testing.T) {
	err := &StructuredOutputError{
		Message: "parse failed",
		RawText: "not json",
	}
	if err.Error() != "parse failed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "parse failed")
	}
}

func TestSkillHelpers(t *testing.T) {
	if !IsSkillFilePath("skills/go.md") {
		t.Error("IsSkillFilePath('skills/go.md') should be true")
	}
	if IsSkillFilePath("go") {
		t.Error("IsSkillFilePath('go') should be false")
	}

	ref := SkillRef{Name: "typescript", Path: "skills/typescript.md"}
	if GetSkillName(ref) != "typescript" {
		t.Errorf("GetSkillName = %q, want %q", GetSkillName(ref), "typescript")
	}
	if GetSkillPath(ref) != "skills/typescript.md" {
		t.Errorf("GetSkillPath = %q, want %q", GetSkillPath(ref), "skills/typescript.md")
	}
}

func TestBuildJsonInstruction(t *testing.T) {
	inst := buildJsonInstruction(JsonRunOptions[json.RawMessage]{
		SchemaName: "Test",
		Schema: map[string]string{
			"name": "string",
		},
	})
	if inst == "" {
		t.Error("buildJsonInstruction returned empty string")
	}
	if !strings.Contains(inst, "Test") {
		t.Error("instruction should contain schema name")
	}
}

func TestCreateDefaultAgentsMd(t *testing.T) {
	content := CreateDefaultAgentsMd("MyProject")
	if !strings.Contains(content, "MyProject") {
		t.Error("default AGENTS.md should contain project name")
	}
	content = CreateDefaultAgentsMd("")
	if !strings.Contains(content, "Project Autopilot") {
		t.Error("default AGENTS.md should contain default title")
	}
}
