# SDLC Workflows With The Go SDK

These workflows use the Go SDK as an orchestration layer around the Autohand CLI.
They mirror the TypeScript and Python SDK examples while using Go channels,
`context.Context`, and typed events.

## Discovery And Planning

Use plan mode when the task is ambiguous:

```go
agent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD:          ".",
    PlanMode:     true,
    Instructions: "Inspect first. Produce a concrete plan before implementation.",
})
if err != nil {
    log.Fatal(err)
}
defer agent.Close()

result, err := agent.Run(ctx, "Plan the smallest safe implementation for this feature.", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text)
```

See `../examples/20-sdlc-discovery-plan`.

## Gated Implementation

Run a read-only pass, review the plan in your host application, then execute only
after an explicit gate:

```go
planAgent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD:      ".",
    PlanMode: true,
})
if err != nil {
    log.Fatal(err)
}

plan, err := planAgent.Run(ctx, "Plan this change without editing files.", nil)
if err != nil {
    log.Fatal(err)
}
_ = planAgent.Close()

if !approvedByHost(plan.Text) {
    return
}

execAgent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD:            ".",
    PermissionMode: autohand.PermissionInteractive,
})
if err != nil {
    log.Fatal(err)
}
defer execAgent.Close()

events, err := execAgent.Stream(ctx, "Implement the approved plan.", nil)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch e := event.(type) {
    case autohand.MessageUpdateEvent:
        fmt.Print(e.Delta)
    case autohand.PermissionRequestEvent:
        _ = execAgent.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce)
    }
}
```

See `../examples/21-sdlc-gated-implementation`.

## Release Readiness

Ask the agent to run the checks that matter for the repository and stream
progress back to the host:

```go
agent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD: ".",
    Instructions: "Report commands run, failures, and residual release risk.",
})
if err != nil {
    log.Fatal(err)
}
defer agent.Close()

events, err := agent.Stream(ctx, `Run release readiness:
- go test ./...
- go vet ./...
- inspect README and examples for drift
`, nil)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch e := event.(type) {
    case autohand.ToolStartEvent:
        fmt.Printf("\n[tool: %s]\n", e.ToolName)
    case autohand.MessageUpdateEvent:
        fmt.Print(e.Delta)
    }
}
```

See `../examples/22-sdlc-release-readiness`.

## Structured Review Output

Use JSON output when another system will consume the result:

```go
data, err := agent.RunJson(ctx, "Return release risk as JSON with summary and risks.", nil,
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

See `../examples/04-structured-json`.
