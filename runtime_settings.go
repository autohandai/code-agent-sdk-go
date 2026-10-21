package autohand

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// buildRuntimeSettings creates a private run-only overlay before the CLI constructs providers.
func buildRuntimeSettings(cfg *Config) (map[string]interface{}, error) {
	if (cfg.Fork || cfg.ForkSession != "") && strings.TrimSpace(forkSessionReference(cfg)) == "" {
		return nil, errors.New("fork requires an explicit session ID or path")
	}
	data, err := json.Marshal(cfg.RunSettings)
	if err != nil {
		return nil, errors.New("run settings must contain JSON values")
	}
	settings := map[string]interface{}{}
	if cfg.RunSettings != nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return nil, errors.New("run settings must be a JSON object")
		}
	}
	if err := validateSettingKeys(settings, 0); err != nil {
		return nil, err
	}
	if cfg.Features != nil {
		data, err := json.Marshal(cfg.Features)
		if err != nil {
			return nil, errors.New("invalid feature settings")
		}
		var flags map[string]interface{}
		if err := json.Unmarshal(data, &flags); err != nil {
			return nil, errors.New("invalid feature settings")
		}
		mergeRuntimeSettings(settings, map[string]interface{}{"features": flags})
	}
	if cfg.Provider == "" {
		if cfg.APIKey != "" || cfg.BaseURL != "" || cfg.Port != 0 || cfg.AutohandAIPlan != "" || cfg.OpenAIAuthMode != "" || cfg.ChatGPTAccessToken != "" || cfg.ChatGPTAccountID != "" || cfg.AzureAuthMethod != "" || cfg.AzureTenantID != "" || cfg.AzureClientID != "" || cfg.AzureClientSecret != "" || cfg.AzureResourceName != "" || cfg.AzureDeploymentName != "" || cfg.AzureAPIVersion != "" {
			return nil, errors.New("provider is required for provider credentials and settings")
		}
		return settings, nil
	}
	provider := string(cfg.Provider)
	section := map[string]interface{}{}
	put := func(key, value string) {
		if value != "" {
			section[key] = value
		}
	}
	put("model", cfg.Model)
	if cfg.APIKey != "" {
		if provider == "vertexai" {
			put("authToken", cfg.APIKey)
		} else {
			put("apiKey", cfg.APIKey)
		}
		switch provider {
		case "autohandai", "openai", "xai":
			section["authMode"] = "api-key"
		case "azure":
			section["authMethod"] = "api-key"
		case "bedrock":
			section["authMode"] = "bedrock-api-key"
		}
	}
	if cfg.BaseURL != "" {
		switch provider {
		case "vertexai":
			value := cfg.BaseURL
			if !strings.Contains(value, "://") {
				value = "https://" + value
			}
			endpoint, err := url.Parse(value)
			if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.Port() != "" || (endpoint.Path != "" && endpoint.Path != "/") || endpoint.RawQuery != "" || endpoint.Fragment != "" {
				return nil, errors.New("Vertex endpoint must be an HTTPS origin or hostname")
			}
			section["endpoint"] = endpoint.Hostname()
		case "bedrock":
			section["endpoint"] = cfg.BaseURL
		default:
			section["baseUrl"] = cfg.BaseURL
		}
	} else if cfg.Port != 0 && (provider == "ollama" || provider == "llamacpp" || provider == "mlx") {
		section["baseUrl"] = "http://localhost:" + strconv.Itoa(cfg.Port)
	}
	if provider == "autohandai" {
		if cfg.AutohandAIPlan != "" {
			section["plan"] = cfg.AutohandAIPlan
		} else if cfg.APIKey != "" || cfg.BaseURL != "" {
			existing, _ := settings[provider].(map[string]interface{})
			if _, ok := existing["plan"]; !ok {
				section["plan"] = "cloud"
			}
		}
	}
	if provider == "openai" {
		put("authMode", cfg.OpenAIAuthMode)
		put("chatgptAccessToken", cfg.ChatGPTAccessToken)
		put("chatgptAccountId", cfg.ChatGPTAccountID)
	}
	if provider == "azure" {
		put("authMethod", cfg.AzureAuthMethod)
		put("tenantId", cfg.AzureTenantID)
		put("clientId", cfg.AzureClientID)
		put("clientSecret", cfg.AzureClientSecret)
		put("resourceName", cfg.AzureResourceName)
		put("deploymentName", cfg.AzureDeploymentName)
		put("apiVersion", cfg.AzureAPIVersion)
	}
	if len(section) > 0 {
		overlay := map[string]interface{}{provider: section}
		if strings.HasPrefix(provider, "custom:") {
			overlay = map[string]interface{}{"customProviders": map[string]interface{}{strings.TrimPrefix(provider, "custom:"): section}}
		} else if strings.HasPrefix(provider, "extension:") {
			overlay = map[string]interface{}{"extensionProviders": map[string]interface{}{provider: section}}
		}
		mergeRuntimeSettings(settings, overlay)
	}
	return settings, nil
}

func mergeRuntimeSettings(target, source map[string]interface{}) {
	for key, value := range source {
		child, object := value.(map[string]interface{})
		existing, existingObject := target[key].(map[string]interface{})
		if object && existingObject {
			mergeRuntimeSettings(existing, child)
		} else {
			target[key] = value
		}
	}
}

func validateSettingKeys(value interface{}, depth int) error {
	if depth > 256 {
		return errors.New("run settings exceed maximum nesting depth")
	}
	switch value := value.(type) {
	case map[string]interface{}:
		for key, item := range value {
			if key == "__proto__" || key == "prototype" || key == "constructor" {
				return errors.New("run settings contain an unsafe setting path")
			}
			if err := validateSettingKeys(item, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range value {
			if err := validateSettingKeys(item, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func needsRuntimeSettings(cfg *Config) bool {
	return cfg.Features != nil || cfg.RunSettings != nil || (cfg.Provider != "" && cfg.Provider != ProviderAutohandAI && (cfg.APIKey != "" || cfg.BaseURL != "" || cfg.Port != 0 || cfg.OpenAIAuthMode != "" || cfg.ChatGPTAccessToken != "" || cfg.ChatGPTAccountID != "" || cfg.AzureAuthMethod != "" || cfg.AzureTenantID != "" || cfg.AzureClientID != "" || cfg.AzureClientSecret != "" || cfg.AzureResourceName != "" || cfg.AzureDeploymentName != "" || cfg.AzureAPIVersion != ""))
}

func forkSessionReference(cfg *Config) string {
	if cfg.ForkSession != "" {
		return cfg.ForkSession
	}
	if cfg.Fork {
		if cfg.SessionID != "" {
			return cfg.SessionID
		}
		return cfg.SessionPath
	}
	return ""
}
