# Plan Mode

Plan mode asks the CLI to restrict the agent to read-only planning behavior. Use
it when a host wants discovery and design before any file writes or commands.

## Enable At Startup

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:      ".",
    PlanMode: true,
})

if err := sdk.Start(ctx); err != nil {
    log.Fatal(err)
}
```

## Toggle At Runtime

```go
if err := sdk.EnablePlanMode(ctx); err != nil {
    log.Fatal(err)
}

if err := sdk.DisablePlanMode(ctx); err != nil {
    log.Fatal(err)
}

if err := sdk.SetPlanMode(ctx, false); err != nil {
    log.Fatal(err)
}
```

`Agent` exposes the same controls:

```go
err := agent.EnablePlanMode(ctx)
err = agent.DisablePlanMode(ctx)
```

## Two-Phase Workflow

```go
planAgent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD:      ".",
    PlanMode: true,
})
if err != nil {
    log.Fatal(err)
}

plan, err := planAgent.Run(ctx, "Plan a refactor. Do not change files.", nil)
if err != nil {
    log.Fatal(err)
}
_ = planAgent.Close()

fmt.Println(plan.Text)

execAgent, err := autohand.NewAgent(ctx, &autohand.Config{
    CWD:            ".",
    PermissionMode: autohand.PermissionInteractive,
})
if err != nil {
    log.Fatal(err)
}
defer execAgent.Close()

result, err := execAgent.Run(ctx, "Implement the approved refactor plan.", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Text)
```

## Plan Mode And Permissions

Plan mode controls what tools are available. Permission mode controls whether the
CLI asks before a tool runs. For review-first workflows, use plan mode for the
first pass and interactive permissions for implementation.
