package autohand

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCurrentHarnessDiscoversEffectiveAgents(t *testing.T) {
	cli := os.Getenv("AUTOHAND_TEST_CLI_PATH")
	if cli == "" {
		t.Skip("set AUTOHAND_TEST_CLI_PATH for current harness integration")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/me" {
			http.Error(w, "no inference expected", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authenticated":true,"user":{"id":"fixture","email":"sdk@example.test"}}`))
	}))
	defer server.Close()
	workspace, home, external := t.TempDir(), t.TempDir(), t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(home, "agents", "user-helper.json"), `{"description":"User helper","systemPrompt":"private user instructions","tools":["read_file"]}`)
	write(filepath.Join(external, "external-helper.json"), `{"description":"External helper","systemPrompt":"private external instructions","tools":["read_file"]}`)
	extension := filepath.Join(workspace, ".autohand", "extensions", "sdk.fixture")
	write(filepath.Join(extension, "autohand.extension.json"), `{"schemaVersion":1,"extensionApi":1,"id":"sdk.fixture","name":"SDK fixture","version":"1.0.0","description":"Discovery fixture","contributes":{"agents":["agents/extension-helper.md"]}}`)
	write(filepath.Join(extension, "agents", "extension-helper.md"), "---\ndescription: Extension helper\ntools: read_file\n---\nPrivate package instructions.\n")
	settings, err := json.Marshal(map[string]interface{}{
		"provider": "autohandai", "auth": map[string]string{"token": "fixture-key"},
		"autohandai":     map[string]interface{}{"model": "fantail", "plan": "cloud", "authMode": "api-key", "contextWindow": 200000},
		"externalAgents": map[string]interface{}{"enabled": true, "paths": []string{external}},
		"features":       map[string]bool{"autohand_inference": true, "automaticSpecialists": false},
		"telemetry":      map[string]bool{"enabled": false},
	})
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(home, "config.json")
	write(config, string(settings))
	sdk := NewSDK(&Config{
		CLIPath: cli, CWD: workspace, Provider: ProviderAutohandAI, APIKey: "fixture-key", BaseURL: server.URL,
		Unrestricted: true, Timeout: 30000, ExtraArgs: []string{"--config", config},
		Agents: `{"inline-helper":{"description":"Inline helper","prompt":"Private inline instructions","model":"fantail","tools":["read_file"]}}`,
		Env:    map[string]string{"AUTOHAND_HOME": home, "AUTOHAND_API_URL": server.URL, "AUTOHAND_AUTH_API_URL": server.URL + "/auth", "AUTOHAND_SKIP_PING": "1", "AUTOHAND_SKIP_UPDATE_CHECK": "1", "AUTOHAND_NO_IDLE_LOGOUT": "1", "AUTOHAND_DISABLE_AUTO_REPORT": "1"},
	})
	defer sdk.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	readAgents := func() map[string]AgentInfo {
		t.Helper()
		agents, err := sdk.SupportedAgents(ctx)
		if err != nil {
			t.Fatal(err)
		}
		byName := make(map[string]AgentInfo, len(agents))
		for _, agent := range agents {
			byName[agent.Name] = agent
		}
		return byName
	}
	agents := readAgents()
	for name, source := range map[string]string{"reviewer": "builtin", "user-helper": "user", "external-helper": "external", "inline-helper": "session", "extension-helper": "extension"} {
		if agent := agents[name]; agent.ID != name || agent.Source != source {
			t.Fatalf("missing effective %s agent %s: %+v", source, name, agent)
		}
	}
	if agents["inline-helper"].Model != "fantail" || agents["extension-helper"].ExtensionID != "sdk.fixture" || agents["extension-helper"].ExtensionScope != "project" {
		t.Fatal("agent metadata was not preserved")
	}
	runCommand := func(message, expected string) {
		t.Helper()
		events, err := sdk.StreamPrompt(ctx, &PromptParams{Message: message})
		if err != nil {
			t.Fatal(err)
		}
		var output strings.Builder
		for event := range events {
			switch event := event.(type) {
			case MessageUpdateEvent:
				output.WriteString(event.Delta)
			case ErrorEvent:
				t.Fatalf("command failed: %+v", event)
			}
		}
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("command output %q does not contain %q", output.String(), expected)
		}
	}
	runCommand("/extensions disable sdk.fixture --scope project", "Disabled sdk.fixture")
	if _, exists := readAgents()["extension-helper"]; exists {
		t.Fatal("disabled extension agent remains available for delegation")
	}
	runCommand("/extensions enable sdk.fixture --scope project", "Enabled sdk.fixture")
	agents = readAgents()
	if agents["extension-helper"].Source != "extension" || agents["inline-helper"].Source != "session" {
		t.Fatal("extension refresh lost effective agent definitions")
	}
	if err := sdk.Prompt(ctx, &PromptParams{Message: "/extensions disable sdk.fixture --scope project"}); err != nil {
		t.Fatal(err)
	}
	if _, exists := readAgents()["extension-helper"]; exists {
		t.Fatal("plain Prompt returned before the extension change finished")
	}
}
