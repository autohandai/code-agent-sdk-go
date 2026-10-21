package autohand

import (
	"context"
	"reflect"
	"testing"
)

func TestRuntimeSettingsProviderFields(t *testing.T) {
	cases := []struct {
		provider                  ProviderName
		key, endpoint, auth, mode string
	}{
		{"anthropic", "apiKey", "baseUrl", "", ""}, {"openai", "apiKey", "baseUrl", "authMode", "api-key"},
		{"xai", "apiKey", "baseUrl", "authMode", "api-key"}, {"azure", "apiKey", "baseUrl", "authMethod", "api-key"},
		{"bedrock", "apiKey", "endpoint", "authMode", "bedrock-api-key"}, {"vertexai", "authToken", "endpoint", "", ""},
	}
	for _, tc := range cases {
		t.Run(string(tc.provider), func(t *testing.T) {
			got, err := buildRuntimeSettings(&Config{Provider: tc.provider, APIKey: "secret", BaseURL: "https://fixture.test"})
			if err != nil {
				t.Fatal(err)
			}
			section := got[string(tc.provider)].(map[string]interface{})
			expected := map[string]interface{}{tc.key: "secret", tc.endpoint: "https://fixture.test"}
			if tc.provider == "vertexai" {
				expected[tc.endpoint] = "fixture.test"
			}
			if tc.auth != "" {
				expected[tc.auth] = tc.mode
			}
			if !reflect.DeepEqual(section, expected) {
				t.Fatalf("fields: %v", section)
			}
		})
	}
}

func TestRuntimeSettingsPreserveDefaultsAndJSON(t *testing.T) {
	no := false
	cfg := &Config{Provider: ProviderAutohandAI, Features: &FeatureFlagSettings{SlashGoal: &no}, RunSettings: map[string]interface{}{"customProviders": map[string]interface{}{"my.provider": map[string]interface{}{"models": []interface{}{"fixture", nil}}}}}
	got, err := buildRuntimeSettings(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got["autohandai"] != nil {
		t.Fatal("omitted provider fields must preserve saved authentication, model and plan")
	}
	if got["features"].(map[string]interface{})["slashGoal"] != false {
		t.Fatal("explicit false lost")
	}
	for _, env := range buildCLIEnv(cfg, nil) {
		if env == "AUTOHAND_AI_PLAN=cloud" {
			t.Fatal("implicit plan override")
		}
	}
	for _, provider := range []ProviderName{"custom:my.provider", "extension:team.provider:model"} {
		result, err := buildRuntimeSettings(&Config{Provider: provider, APIKey: "secret"})
		if err != nil {
			t.Fatal(err)
		}
		if provider == "custom:my.provider" && result["customProviders"].(map[string]interface{})["my.provider"] == nil {
			t.Fatal(result)
		}
		if provider == "extension:team.provider:model" && result["extensionProviders"].(map[string]interface{})[string(provider)] == nil {
			t.Fatal(result)
		}
	}
}

func TestRuntimeSettingsFailBeforeCLIDiscovery(t *testing.T) {
	cfg := &Config{APIKey: "secret"}
	err := NewTransport(cfg).Start(context.Background(), cfg)
	if err == nil || err.Error() != "provider is required for provider credentials and settings" {
		t.Fatalf("error: %v", err)
	}
	for _, settings := range []map[string]interface{}{{"unsafe": make(chan int)}, {"customProviders": map[string]interface{}{"fixture": []interface{}{map[string]interface{}{"__proto__": true}}}}} {
		if _, err := buildRuntimeSettings(&Config{RunSettings: settings}); err == nil {
			t.Fatal("unsafe setting accepted")
		}
	}
}

func TestCurrentSessionArguments(t *testing.T) {
	cases := []struct {
		cfg  Config
		want []string
	}{
		{Config{SessionID: "saved", PersistSession: true}, []string{"--mode", "rpc", "--sdk-session-reference", "saved"}},
		{Config{SessionPath: "/saved/path", Resume: true}, []string{"--mode", "rpc", "--sdk-session-reference", "/saved/path"}},
		{Config{Continue: true}, []string{"--mode", "rpc", "--sdk-resume-last"}},
		{Config{Fork: true, SessionID: "saved"}, []string{"--mode", "rpc", "--fork", "saved"}},
	}
	for _, tc := range cases {
		if got := buildCLIArgs(&tc.cfg); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("args: %v want %v", got, tc.want)
		}
	}
	if _, err := buildRuntimeSettings(&Config{Fork: true}); err == nil {
		t.Fatal("fork requires an explicit source")
	}
}

func TestClearThinkingBudgetUsesExplicitNull(t *testing.T) {
	client, requests, closeTransport := newRPCTestClient(t)
	defer closeTransport()
	if err := client.ClearMaxThinkingTokens(context.Background()); err != nil {
		t.Fatal(err)
	}
	request := <-requests
	if string(request.Params) != "{\"maxThinkingTokens\":null}" {
		t.Fatalf("params: %s", request.Params)
	}
}
