# Error Handling

The Go SDK reports failures as ordinary `error` values and agent-loop failures
as typed `ErrorEvent` values in the event stream.

## Transport Errors

Transport errors happen when the CLI cannot be detected, cannot start, exits, or
stops responding.

```go
sdk := autohand.NewSDK(&autohand.Config{
    CLIPath: "/path/to/autohand",
    CWD:     ".",
})

if err := sdk.Start(ctx); err != nil {
    log.Fatalf("start Autohand CLI: %v", err)
}
```

Common causes:

- `CLIPath` points at a missing or non-executable file.
- The bundled platform binary is not present.
- The CLI provider config is missing or invalid.
- The working directory does not exist.

## JSON-RPC Errors

RPC errors come from the CLI JSON-RPC layer. They are returned by methods such
as `Prompt`, `SetModel`, `SetPlanMode`, and `PermissionResponse`.

```go
if err := sdk.SetModel(ctx, "openrouter/auto"); err != nil {
    log.Printf("model switch failed: %v", err)
}
```

The default interactive permission mode is handled by the CLI startup path. The
SDK only calls the permission-mode RPC for non-default modes.

## Agent Events

The agent can emit errors while a prompt is running:

```go
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: "Run tests"})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    if e, ok := event.(autohand.ErrorEvent); ok {
        log.Printf("agent error %d: %s", e.Code, e.Message)
    }
}
```

These often represent provider failures, tool failures, or prompt-time RPC
errors surfaced through the stream.

## Timeouts And Cancellation

Set the SDK timeout in milliseconds:

```go
sdk := autohand.NewSDK(&autohand.Config{
    Timeout: 60000,
})
```

Use `context.Context` cancellation to stop waiting from the host side:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

result, err := agent.Run(ctx, "Summarize this repo", nil)
if err != nil {
    log.Fatal(err)
}
_ = result
```

Abort the current agent turn:

```go
if err := sdk.Abort(ctx); err != nil {
    log.Printf("abort failed: %v", err)
}
```

## Structured Output Errors

JSON helpers return `StructuredOutputError` when parsing or validation fails:

```go
data, err := agent.RunJson(ctx, "Return JSON", nil)
if err != nil {
    var structured *autohand.StructuredOutputError
    if errors.As(err, &structured) {
        log.Printf("raw response: %s", structured.RawText)
    }
    log.Fatal(err)
}
fmt.Println(string(data))
```

## Recovery Pattern

For long-lived hosts:

1. Start the SDK with a bounded timeout.
2. Treat transport errors as session failures and recreate the SDK.
3. Treat `ErrorEvent` as run failures and decide whether to retry the prompt.
4. Always call `Close` when the host no longer needs the subprocess.
