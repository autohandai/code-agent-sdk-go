# Configuration

The SDK accepts a single `Config` object. Every field is optional.

## Basic Options

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:     ".",                    // Working directory
    CLIPath: "/path/to/cli",         // Custom CLI binary
    Debug:   true,                   // Log JSON-RPC traffic
    Timeout: 30000,                  // Request timeout in ms
})
```

## Provider Setup

The SDK delegates LLM calls to the CLI, so provider credentials live in `~/.autohand/config.json`:

```json
{
  "provider": "openrouter",
  "openrouter": {
    "apiKey": "sk-or-...",
    "model": "openrouter/auto"
  }
}
```

You can override the model at runtime:

```go
sdk.SetModel(ctx, "openrouter/auto")
```

### Supported Providers

| Provider | Notes |
|---|---|
| OpenRouter | Set `apiKey` and optional `model` in CLI config. |
| OpenAI | Set `apiKey` or use `ChatGPTAccessToken`. |
| Azure | Needs `AzureAuthMethod`, `AzureTenantID`, etc. |
| Ollama | Local. Set `Port` if not running on 11434. |
| LlamaCPP | Local. Set `Port`. |
| MLX | Local. Set `Port`. |

The SDK auto-detects the provider from the model string when possible. Pass `Provider` explicitly if auto-detection fails.

## Execution Mode

```go
sdk := autohand.NewSDK(&autohand.Config{
    AutoMode:      true,   // Let the agent run autonomously
    MaxIterations: 10,     // Max auto-mode turns
    MaxRuntime:    30,     // Max runtime in minutes
    MaxCost:       5.0,    // Max API cost in USD
})
```

## Skills

```go
sdk := autohand.NewSDK(&autohand.Config{
    AutoSkill: true,
    Skills: []autohand.SkillRef{
        {Name: "typescript"},
        {Name: "react"},
    },
})
```

## Context

```go
sdk := autohand.NewSDK(&autohand.Config{
    ContextCompact:         true,
    MaxTokens:              128000,
    CompressionThreshold:   0.7,
    SummarizationThreshold: 0.9,
})
```

## Session Persistence

```go
sdk := autohand.NewSDK(&autohand.Config{
    PersistSession:   true,
    Resume:           false,
    SessionPath:      "./.autohand/sessions",
    AutoSaveInterval: 60,
})
```

## System Prompts

```go
sdk := autohand.NewSDK(&autohand.Config{
    SysPrompt:       "You are a careful code reviewer.",
    AppendSysPrompt: "Always run tests before declaring a task done.",
})
```

Both accept inline strings. The SDK passes them to the CLI as flags.

## AGENTS.md

```go
sdk := autohand.NewSDK(&autohand.Config{
    AgentsMdEnable:     true,
    AgentsMdCreate:     true,
    AgentsMdPath:       "./AGENTS.md",
    AgentsMdAutoUpdate: true,
})
```

## Full Example

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:         ".",
    Model:       "openrouter/auto",
    Temperature: 0.7,
    Debug:       true,
    AutoMode:    true,
    MaxIterations: 10,
    PermissionMode: autohand.PermissionInteractive,
    Skills: []autohand.SkillRef{
        {Name: "typescript"},
    },
    ContextCompact: true,
    MaxTokens:      128000,
    AppendSysPrompt: "Always write tests for new code.",
})
```
