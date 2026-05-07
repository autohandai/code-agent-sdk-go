# Getting Started with the Autohand Go SDK

The Autohand Go SDK is a thin wrapper around the Autohand CLI. It spawns the CLI as a subprocess and talks to it over JSON-RPC, giving you a typed, programmatic interface to an autonomous coding agent.

## Prerequisites

1. The Autohand CLI binary. The SDK ships with prebuilt binaries for macOS, Linux, and Windows, but you can also point to a custom build.
2. A provider API key. The SDK delegates all LLM calls to the CLI, so you configure the provider in the CLI config file (`~/.autohand/config.json`) rather than in the SDK itself.

Example `~/.autohand/config.json`:

```json
{
  "provider": "openrouter",
  "openrouter": {
    "apiKey": "sk-or-...",
    "model": "openrouter/auto"
  }
}
```

## Installation

```bash
go get github.com/autohandai/code-agent-sdk-go
```

## Your First Prompt

Create a file named `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
    ctx := context.Background()
    sdk := autohand.NewSDK(&autohand.Config{
        CWD:   ".",
        Debug: true,
    })

    if err := sdk.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer sdk.Close()

    if err := sdk.Prompt(ctx, &autohand.PromptParams{
        Message: "List the Go files in the current directory",
    }); err != nil {
        log.Fatal(err)
    }
}
```

Run it:

```bash
go run main.go
```

The SDK will auto-detect the correct CLI binary for your platform, spawn it, send the prompt, and shut it down when you call `Close()`.

## Streaming Events

`Prompt()` is fire-and-forget. Most applications want to see what the agent is doing in real time. Use `StreamPrompt()` instead:

```go
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
    Message: "What does main.go do?",
})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch e := event.(type) {
    case autohand.MessageUpdateEvent:
        fmt.Print(e.Delta)
    case autohand.ToolStartEvent:
        fmt.Printf("\n[tool: %s]\n", e.ToolName)
    case autohand.ToolEndEvent:
        fmt.Printf("[tool completed: %s]\n", e.ToolName)
    }
}
```

The event stream includes message deltas, tool calls, tool outputs, permission requests, and errors. You decide which ones to surface to the user.

## Handling Permissions

By default the CLI asks before running shell commands or making file changes. In `StreamPrompt()` these show up as `PermissionRequestEvent`:

```go
for event := range events {
    if e, ok := event.(autohand.PermissionRequestEvent); ok {
        fmt.Printf("\nPermission requested: %s\n", e.Tool)
        fmt.Printf("Description: %s\n", e.Description)

        // Approve this request
        if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
            log.Printf("allow: %v", err)
        }
    }
}
```

For unattended scripts you can disable interactive permission checks with `PermissionMode: autohand.PermissionUnrestricted`, though use that with caution.

## Using the High-Level Agent API

If you do not want to manage the subprocess lifecycle manually, use the `Agent` type:

```go
agent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD:            ".",
    PermissionMode: autohand.PermissionInteractive,
})
if err != nil {
    log.Fatal(err)
}
defer agent.Close()

run, err := agent.Send(ctx, "Review this repository for release readiness", nil)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    if e, ok := event.(autohand.MessageUpdateEvent); ok {
        fmt.Print(e.Delta)
    }
}

result, err := run.Wait(ctx)
fmt.Println(result.Text)
```

`NewAgent()` handles `Start()` for you. `agent.Close()` stops the CLI. A single `Agent` instance can handle multiple sequential runs.

## Next Steps

- See the `examples/` directory for complete, runnable programs covering streaming, permissions, structured JSON output, and more.
- Read `docs/configuration.md` for all configuration options.
- Read `docs/event-streaming.md` for a complete event type reference.
