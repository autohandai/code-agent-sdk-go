package autohand

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Set AUTOHAND_TEST_CLI_PATH to a current CLI executable. HTTP stays local;
// this exercises the real harness/provider/tool/step-persistence boundary.
func TestCurrentHarnessStopWhenWithAutohandAI(t *testing.T) {
	cli := os.Getenv("AUTOHAND_TEST_CLI_PATH")
	if cli == "" {
		t.Skip("set AUTOHAND_TEST_CLI_PATH for current harness integration")
	}
	var calls atomic.Int32
	var resumedEvidence atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/me" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"authenticated":true,"user":{"id":"fixture","email":"sdk@example.test","name":"SDK Fixture"}}`))
			return
		}
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer sdk-fixture-key" {
			http.Error(w, "unexpected provider request", http.StatusBadRequest)
			return
		}
		var request struct {
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		message := map[string]interface{}{"role": "assistant", "content": "continued from persisted evidence"}
		if calls.Add(1) == 1 {
			message["content"] = "Inspect the evidence file."
			message["tool_calls"] = []interface{}{map[string]interface{}{
				"id": "call-read", "type": "function", "function": map[string]interface{}{"name": "read_file", "arguments": `{"path":"evidence.txt"}`},
			}}
		} else {
			for _, entry := range request.Messages {
				if strings.Contains(entry.Content, "sdk-parity-evidence") {
					resumedEvidence.Store(true)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "fixture", "choices": []interface{}{map[string]interface{}{"message": message, "finish_reason": "stop"}},
			"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 10, "total_tokens": 20},
		}); err != nil {
			t.Errorf("write provider response: %v", err)
		}
	}))
	defer server.Close()
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "evidence.txt"), []byte("sdk-parity-evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(workspace, "config.json")
	settings, err := json.Marshal(map[string]interface{}{
		"auth":     map[string]string{"token": "sdk-fixture-key"},
		"provider": "openrouter", "openrouter": map[string]string{"baseUrl": server.URL + "/unused", "apiKey": "saved-provider-key"},
		"autohandai": map[string]interface{}{"model": "fantail", "plan": "cloud", "authMode": "api-key", "contextWindow": 200000},
		"features":   map[string]bool{"autohand_inference": true, "automaticSpecialists": false},
		"telemetry":  map[string]bool{"enabled": false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, settings, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	agent, err := NewAgent(ctx, &Config{
		CLIPath: cli, CWD: workspace, Provider: ProviderAutohandAI, APIKey: "sdk-fixture-key", BaseURL: server.URL,
		Model: "fantail", Bare: true, Unrestricted: true, Timeout: 30000, ExtraArgs: []string{"--config", config},
		Env: map[string]string{"AUTOHAND_HOME": t.TempDir(), "AUTOHAND_API_KEY": "sdk-fixture-key", "AUTOHAND_API_URL": server.URL, "AUTOHAND_AUTH_API_URL": server.URL + "/auth", "AUTOHAND_SKIP_PING": "1", "AUTOHAND_SKIP_UPDATE_CHECK": "1", "AUTOHAND_NO_IDLE_LOGOUT": "1", "AUTOHAND_DISABLE_AUTO_REPORT": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer agent.Close()
	condition, err := IsStepCount(1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := agent.Run(ctx, "Read evidence.txt with read_file.", &PromptParams{StopWhen: []StopCondition{condition}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "stopped" || len(result.Steps) != 1 || calls.Load() != 1 {
		t.Fatalf("stop: %+v, requests=%d", result, calls.Load())
	}
	if len(result.Steps[0].ToolResults) != 1 || !result.Steps[0].ToolResults[0].Success || !strings.Contains(result.Steps[0].ToolResults[0].Output, "sdk-parity-evidence") {
		t.Fatalf("read_file result was not persisted: %+v", result.Steps[0])
	}
	result, err = agent.Run(ctx, "Continue using the saved tool result.", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || !resumedEvidence.Load() {
		t.Fatalf("continuation: %+v; persisted evidence=%v", result, resumedEvidence.Load())
	}
	savedBytes, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Provider   string
		AutohandAI map[string]interface{}
		OpenRouter map[string]string
	}
	if err := json.Unmarshal(savedBytes, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Provider != "openrouter" || saved.OpenRouter["apiKey"] != "saved-provider-key" || saved.AutohandAI["apiKey"] != nil || saved.AutohandAI["baseUrl"] != nil {
		t.Fatal("process provider settings leaked into the saved configuration")
	}
}
