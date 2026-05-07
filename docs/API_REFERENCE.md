# API Reference

Complete reference for the Go Autohand SDK.

## Package

```go
import autohand "github.com/autohandai/code-agent-sdk-go"
```

The package exposes two layers:

- `Agent` and `Run` for application-level orchestration.
- `SDK`, `RPCClient`, and `Transport` for hosts that need lower-level control.

## High-Level API

### NewAgent

```go
agent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD: ".",
    Instructions: "Prefer small, tested changes.",
})
```

`NewAgent` creates an `SDK`, starts the CLI subprocess, and returns an `Agent`.
Call `agent.Close()` when finished.

### Agent.Send

```go
run, err := agent.Send(ctx, "Review this repository", nil)
```

Creates a `Run` without waiting for completion. Use this when you want to stream
events and then await the final result.

### Agent.Run

```go
result, err := agent.Run(ctx, "Summarize release readiness", nil)
fmt.Println(result.Text)
```

Runs a prompt to completion and returns `RunResult`.

### Agent.Stream

```go
events, err := agent.Stream(ctx, "Explain this package", nil)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    if e, ok := event.(autohand.MessageUpdateEvent); ok {
        fmt.Print(e.Delta)
    }
}
```

Streams events for a single prompt.

### Agent.RunJson

```go
data, err := agent.RunJson(ctx, "Return {\"ok\": true}", nil)
```

Runs a prompt with JSON instructions and parses the final response as
`json.RawMessage`. Responses may be direct JSON, fenced JSON, or embedded JSON.
Parse failures return `StructuredOutputError` with the raw text preserved.

### Run

```go
events, err := run.Stream(ctx)
result, err := run.Wait(ctx)
err = run.Abort(ctx)
data, err := run.JSON()
```

`Run.Stream` replays events already seen and follows new events until completion.
`Run.Wait` returns the final `RunResult`.

## Low-Level API

### NewSDK

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD: ".",
    Debug: true,
})
```

Creates an SDK instance. It does not start the CLI until `Start`, `Prompt`, or
another method that calls `ensureStarted`.

### Lifecycle

```go
err := sdk.Start(ctx)
err = sdk.Stop()
err = sdk.Close()
```

`Close` is an alias for `Stop`.

### Prompting

```go
err := sdk.Prompt(ctx, &autohand.PromptParams{Message: "Hello"})
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: "Hello"})
```

`Prompt` sends a prompt and waits for the RPC request to settle. `StreamPrompt`
returns a channel of typed events.

### Runtime Controls

```go
err := sdk.SetModel(ctx, "openrouter/auto")
err = sdk.SetPlanMode(ctx, true)
err = sdk.EnablePlanMode(ctx)
err = sdk.DisablePlanMode(ctx)
err = sdk.SetPermissionMode(ctx, autohand.PermissionRestricted)
err = sdk.SetMaxThinkingTokens(ctx, 4096)
err = sdk.ApplyFlagSettings(ctx, map[string]interface{}{"temperature": 0.2})
err = sdk.Abort(ctx)
```

`Start` skips the permission-mode RPC for the default interactive mode.
Non-default permission modes call the CLI RPC method.

### State And Metadata

```go
state, err := sdk.GetState(ctx)
messages, err := sdk.GetMessages(ctx, 20)
models, err := sdk.SupportedModels(ctx)
usage, err := sdk.GetContextUsage(ctx)
account, err := sdk.AccountInfo(ctx)
stats, err := sdk.GetStats(ctx)
metadata, err := sdk.GetSessionMetadata(ctx)
```

### Permissions

```go
err := sdk.AllowPermission(ctx, requestID, autohand.ScopeOnce)
err = sdk.DenyPermission(ctx, requestID, autohand.ScopeSession)
err = sdk.PermissionResponse(ctx, requestID, autohand.DecisionAlternative)
```

### Hooks And MCP

```go
hooks, err := sdk.GetHooks(ctx)
err = sdk.AddHook(ctx, autohand.HookDefinition{Event: "pre-tool", Command: "echo {{tool}}"})
err = sdk.ToggleHook(ctx, "pre-tool", 0)
err = sdk.RemoveHook(ctx, "pre-tool", 0)
err = sdk.ToggleMCPServer(ctx, "filesystem", true)
err = sdk.SetMCPServers(ctx, map[string]autohand.MCPServerConfig{})
```

### System Prompts And AGENTS.md

```go
err := sdk.SetSystemPrompt(ctx, "./SYSTEM_PROMPT.md")
err = sdk.AppendSystemPrompt(ctx, "Always run go test ./... before summary.")
content, err := sdk.LoadAgentsMd("./AGENTS.md")
params := sdk.SetAgentsMdAsPrompt(ctx, &autohand.PromptParams{Message: "Review"}, content)
```

## Core Types

### Config

`Config` contains startup flags, provider hints, permission settings, context
settings, session settings, hooks, MCP servers, additional directories, and
environment variables.

Common fields:

- `CWD`, `CLIPath`, `Debug`, `Timeout`
- `Model`, `Provider`, `APIKey`, `BaseURL`, `Temperature`
- `PermissionMode`, `Permissions`, `Yolo`, `YoloTimeout`, `PlanMode`
- `AutoMode`, `Unrestricted`, `MaxIterations`, `MaxRuntime`, `MaxCost`
- `SysPrompt`, `AppendSysPrompt`, `Instructions`
- `Skills`, `SkillRefs`
- `ContextCompact`, `MaxTokens`, `CompressionThreshold`, `SummarizationThreshold`
- `PersistSession`, `SessionID`, `Resume`, `Continue`, `SessionPath`
- `AdditionalDirectories`, `AddDir`, `Env`, `EnvVars`, `ExtraArgs`

### PromptParams

```go
type PromptParams struct {
    Message       string
    Context       *PromptContext
    ThinkingLevel string
}
```

`Context` can include files, selected ranges, image attachments, and AGENTS.md
content depending on the CLI capabilities in use.

### RunResult

```go
type RunResult struct {
    ID     string
    Status string
    Text   string
    Events []autohand.Event
}
```

### Event

Events are concrete Go structs that satisfy the `Event` interface. Use a type
switch in application code:

```go
switch e := event.(type) {
case autohand.MessageUpdateEvent:
    fmt.Print(e.Delta)
case autohand.ToolStartEvent:
    fmt.Printf("tool: %s\n", e.ToolName)
case autohand.PermissionRequestEvent:
    _ = sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce)
case autohand.ErrorEvent:
    log.Printf("agent error: %s", e.Message)
}
```

## Event Types

- `AgentStartEvent`
- `AgentEndEvent`
- `TurnStartEvent`
- `TurnEndEvent`
- `MessageStartEvent`
- `MessageUpdateEvent`
- `MessageEndEvent`
- `ToolStartEvent`
- `ToolUpdateEvent`
- `ToolEndEvent`
- `FileModifiedEvent`
- `PermissionRequestEvent`
- `ErrorEvent`

## Error Types

Most methods return ordinary Go `error` values. JSON output parsing returns
`StructuredOutputError`, which includes `RawText` so callers can log or retry
with the model response that failed parsing.
