# Event Streaming

`StreamPrompt()` returns a channel that yields events as they happen. You read them in a `for...range` loop and decide what to show the user.

## Basic Pattern

```go
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: "Hello"})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    if e, ok := event.(autohand.MessageUpdateEvent); ok {
        fmt.Print(e.Delta)
    }
}
```

## Event Types

### MessageUpdateEvent

A chunk of the agent response. Concatenate `Delta` to build the full message.

```go
if e, ok := event.(autohand.MessageUpdateEvent); ok {
    fmt.Print(e.Delta)
}
```

### MessageEndEvent

The agent finished generating. `Content` contains the full message string.

```go
if e, ok := event.(autohand.MessageEndEvent); ok {
    fmt.Println("\n--- done ---")
}
```

### ToolStartEvent

The agent called a tool.

```go
if e, ok := event.(autohand.ToolStartEvent); ok {
    fmt.Printf("[tool: %s]\n", e.ToolName)
}
```

### ToolUpdateEvent

Streaming output from a running tool (stdout or file contents).

```go
if e, ok := event.(autohand.ToolUpdateEvent); ok {
    fmt.Print(e.Output)
}
```

### ToolEndEvent

The tool finished. `Output` may contain the final result.

```go
if e, ok := event.(autohand.ToolEndEvent); ok {
    fmt.Printf("[tool completed: %s]\n", e.ToolName)
    if e.Output != "" {
        fmt.Println(e.Output[:min(len(e.Output), 500)])
    }
}
```

### PermissionRequestEvent

The CLI needs approval before running a tool.

```go
if e, ok := event.(autohand.PermissionRequestEvent); ok {
    fmt.Printf("Permission needed: %s\n", e.Tool)
    fmt.Printf("Description: %s\n", e.Description)

    if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
        log.Printf("allow: %v", err)
    }
}
```

### ErrorEvent

Something went wrong inside the agent loop or transport.

```go
if e, ok := event.(autohand.ErrorEvent); ok {
    log.Printf("Agent error: %s", e.Message)
}
```

## Building a Simple Chat UI

```go
var fullMessage string

for event := range events {
    switch e := event.(type) {
    case autohand.MessageUpdateEvent:
        fmt.Print(e.Delta)
        fullMessage += e.Delta
    case autohand.ToolStartEvent:
        fmt.Printf("\n[running %s]\n", e.ToolName)
    case autohand.ToolEndEvent:
        fmt.Printf("[%s done]\n", e.ToolName)
    case autohand.PermissionRequestEvent:
        isShell := e.Tool == "run_command" || e.Tool == "bash"
        decision := autohand.ScopeOnce
        if isShell {
            decision = autohand.ScopeOnce // or deny
        }
        if err := sdk.AllowPermission(ctx, e.RequestID, decision); err != nil {
            log.Printf("permission: %v", err)
        }
    case autohand.ErrorEvent:
        log.Printf("Error: %s", e.Message)
    }
}
```

## Subscribing to All Events

If you want events outside of a prompt stream:

```go
events, err := sdk.Events(ctx)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    fmt.Printf("event type: %T\n", event)
}
```

This includes lifecycle events like `AgentStartEvent` and `AgentEndEvent`.
