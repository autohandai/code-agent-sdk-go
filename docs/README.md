# Autohand Go SDK Documentation

The Go SDK is a CLI wrapper around Autohand. It starts the Autohand CLI in
JSON-RPC mode, sends prompts, streams typed events, and exposes both a low-level
`SDK` API and a high-level `Agent` / `Run` API.

```text
Go application -> autohand package -> CLI subprocess -> AI provider
```

## Guides

- [Getting Started](./getting-started.md)
- [API Reference](./API_REFERENCE.md)
- [Configuration](./configuration.md)
- [Event Streaming](./event-streaming.md)
- [Error Handling](./error-handling.md)
- [Advanced Patterns](./advanced-patterns.md)
- [Permissions](./permissions.md)
- [Plan Mode](./plan-mode.md)
- [Memory](./memory.md)
- [SDLC Workflows](./sdlc-workflows.md)

## Examples

Runnable programs live in `../examples/`. Each example is its own Go module so
it can be copied into a separate project or run in place during SDK development.
