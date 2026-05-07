# Advanced Patterns

## Custom System Prompts

Replace the entire CLI system prompt before a session starts:

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD: ".",
    SysPrompt: "./SYSTEM_PROMPT.md",
})
```

Append to the default system prompt:

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD: ".",
    AppendSysPrompt: "Always run go test ./... before declaring success.",
})
```

At runtime:

```go
if err := sdk.SetSystemPrompt(ctx, "./SYSTEM_PROMPT.md"); err != nil {
    log.Fatal(err)
}

if err := sdk.AppendSystemPrompt(ctx, "Prefer small, reviewed diffs."); err != nil {
    log.Fatal(err)
}
```

## High-Level Runs

`Agent` is the recommended API for host applications because it keeps run
lifecycle explicit:

```go
agent, err := autohand.NewAgent(ctx, &autohand.Config{CWD: "."})
if err != nil {
    log.Fatal(err)
}
defer agent.Close()

run, err := agent.Send(ctx, "Review this package", nil)
if err != nil {
    log.Fatal(err)
}

events, err := run.Stream(ctx)
if err != nil {
    log.Fatal(err)
}
for event := range events {
    if e, ok := event.(autohand.MessageUpdateEvent); ok {
        fmt.Print(e.Delta)
    }
}

result, err := run.Wait(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Status)
```

## Structured JSON Output

`RunJson` adds JSON instructions, waits for completion, and parses the final
message:

```go
data, err := agent.RunJson(ctx, "Assess this release as JSON", nil,
    autohand.JsonParseOptions[json.RawMessage]{
        Validate: func(value interface{}) (json.RawMessage, error) {
            return json.Marshal(value)
        },
    },
)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

The parser accepts direct JSON, fenced JSON, and embedded JSON. Invalid responses
return `StructuredOutputError` with the raw response attached.

## Hooks

Hooks let the CLI run commands around tool execution when the CLI supports the
corresponding hook RPCs:

```go
err := sdk.AddHook(ctx, autohand.HookDefinition{
    Event:   "pre-tool",
    Command: "echo about to run {{tool}}",
})
```

Inspect or update hooks:

```go
hooks, err := sdk.GetHooks(ctx)
err = sdk.ToggleHook(ctx, "pre-tool", 0)
err = sdk.RemoveHook(ctx, "pre-tool", 0)
_ = hooks
```

## Context Compaction

Pass context settings at startup:

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:                    ".",
    ContextCompact:         true,
    MaxTokens:              128000,
    CompressionThreshold:   0.7,
    SummarizationThreshold: 0.9,
})
```

Inspect current usage:

```go
usage, err := sdk.GetContextUsage(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("messages: %d, memory files: %d\n", usage.Messages, usage.MemoryFiles)
```

## Session Persistence

Let the CLI persist sessions:

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:              ".",
    PersistSession:   true,
    SessionPath:      "./.autohand/sessions",
    AutoSaveInterval: 60,
})
```

Resume an existing session:

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:       ".",
    SessionID: "session-id",
    Resume:    true,
})
```

The Go SDK also exposes `SaveSession`, `ResumeSession`, `GetStats`, and
`GetSessionMetadata` for hosts that need explicit session controls.

## AGENTS.md

Load AGENTS.md content from a file, URL, or inline text:

```go
content, err := sdk.LoadAgentsMd(ctx, "./AGENTS.md")
if err != nil {
    log.Fatal(err)
}

params := sdk.SetAgentsMdAsPrompt(ctx, &autohand.PromptParams{
    Message: "Review this project using its local guidance.",
}, content)

events, err := sdk.StreamPrompt(ctx, params)
_ = events
_ = err
```

Generate a starter file:

```go
template := sdk.CreateDefaultAgentsMd("My Project")
fmt.Println(template)
```
