# Code Agent SDK for Go

Autohand Code Agent SDK - CLI wrapper implementation for Go.

**Beta:** this SDK is actively evolving while the Agent SDK APIs stabilize. Pin versions in production and review release notes before upgrading.

## Overview

This SDK provides a Go wrapper around the Autohand CLI binary, enabling programmatic access to Autohand's autonomous coding agent capabilities via JSON-RPC 2.0 protocol.

## Architecture

```
User -> Go SDK (thin wrapper) -> CLI Subprocess (existing binary) -> Provider -> HTTP
```

The SDK:
- Spawns the Autohand CLI as a subprocess
- Communicates via JSON-RPC 2.0 over stdin/stdout
- Provides an idiomatic Go API
- Supports streaming events

## Other Programming Languages (Beta)

The Agent SDK is available in multiple beta language packages. Use the same CLI-backed SDK model from another programming language:

- [TypeScript](https://github.com/autohandai/code-agent-sdk-typescript) - `Agent`, `Run`, streaming, and JSON helpers for Node and Bun hosts.
- [Go](https://github.com/autohandai/code-agent-sdk-go) - this package, with `context.Context`, typed events, and channel-based streaming.
- [Python](https://github.com/autohandai/code-agent-sdk-python) - async Python package with `async for` event streams and typed Pydantic models.
- [Java](https://github.com/autohandai/code-agent-sdk-java) - Java 21 records, sealed events, and virtual-thread-ready APIs.
- [Swift](https://github.com/autohandai/code-agent-sdk-swift) - SwiftPM package with `Agent`, `Runner`, async streams, tools, hooks, and permissions.

## Installation

```bash
go get github.com/autohandai/code-agent-sdk-go
```

## Quick Start

### High-Level API

Use `Agent` for application code. It gives you an explicit run lifecycle while keeping CLI subprocess and JSON-RPC details out of your app.

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
    agent, err := autohand.NewAgent(ctx, &autohand.Config{
        CWD: ".",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer agent.Close()

    result, err := agent.Run(ctx, "Summarize the API surface", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result.Text)
}
```

For JSON output:

```go
type ReleaseRisk struct {
    Summary string `json:"summary"`
    Risks   []struct {
        Title      string `json:"title"`
        Severity   string `json:"severity"`
        Mitigation string `json:"mitigation"`
    } `json:"risks"`
}

data, err := agent.RunJson(ctx, "Assess publish readiness", nil)
if err != nil {
    log.Fatal(err)
}

var risk ReleaseRisk
if err := json.Unmarshal(data, &risk); err != nil {
    log.Fatal(err)
}
```

### Low-Level API

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:   ".",
    Debug: true,
})

ctx := context.Background()
if err := sdk.Start(ctx); err != nil {
    log.Fatal(err)
}
defer sdk.Close()

// Send a prompt
if err := sdk.Prompt(ctx, &autohand.PromptParams{
    Message: "Hello, Autohand!",
}); err != nil {
    log.Fatal(err)
}

// Stream events
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
    Message: "Analyze the codebase",
})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    fmt.Printf("%+v\n", event)
}
```

## Configuration

See `docs/configuration.md` for all options.

## API Reference

See the `docs/` directory:

- `docs/getting-started.md` - installation, first prompt, streaming, and high-level API.
- `docs/API_REFERENCE.md` - public types and methods.
- `docs/configuration.md` - CLI, provider, execution, skills, context, and session options.
- `docs/event-streaming.md` - event types and channel patterns.
- `docs/error-handling.md` - transport, RPC, timeout, and recovery patterns.
- `docs/advanced-patterns.md` - system prompts, hooks, JSON output, sessions, and AGENTS.md.
- `docs/permissions.md` - permission modes and programmatic approval.
- `docs/plan-mode.md` - read-only planning and gated implementation.
- `docs/memory.md` - CLI memory behavior through SDK event streams.
- `docs/sdlc-workflows.md` - discovery, gated implementation, and release-readiness flows.

## Examples

See the `examples/` directory for runnable programs:
- `01-hello-agent` - Basic prompt
- `02-streaming-query` - Event streaming
- `03-high-level-agent` - Agent API
- `04-structured-json` - JSON output

## Development

```bash
# Build
go build ./...

# Test
go test ./...

# Vet
go vet ./...
```

## License

MIT
