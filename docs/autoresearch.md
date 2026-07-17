# Replayable Autoresearch Ledger

The Go SDK exposes Autohand's persisted autoresearch engine through typed
JSON-RPC methods. A session proposes focused changes, evaluates each candidate,
records immutable measurements and decisions under `.auto/`, and can replay or
rescore earlier attempts without losing their original evidence.

Use a CLI build that supports autoresearch RPC methods. When CLI versions may
vary, check `SDK.SupportedAgents` or the low-level supported-command surface
before presenting autoresearch in your application.

## Start or resume

```go
started, err := agent.StartAutoresearch(ctx, &autohand.AutoresearchStartParams{
    Objective:      "Reduce checkout p95 latency without changing behavior",
    MaxIterations:  12,
    MetricName:     "p95_ms",
    MetricUnit:     "ms",
    Direction:      autohand.AutoresearchLower,
    MeasureCommand: "go test ./... -run BenchmarkCheckout -bench . -count 5",
    ChecksCommand:  "go test ./...",
    FilesInScope:   []string{"checkout/", "internal/cache/"},
    Sampling: &autohand.AutoresearchSamplingOptions{
        MinSamples: 3,
        MaxSamples: 7,
        ConfidenceThreshold: 0.9,
    },
    Constraints: []autohand.AutoresearchConstraint{{
        MetricName: "error_rate",
        Operator:   autohand.AutoresearchConstraintLessThanOrEqual,
        Threshold:  0,
    }},
})
if err != nil {
    return err
}
if !started.Success {
    return fmt.Errorf("start autoresearch: %s", started.Error)
}
```

Calling `StartAutoresearch` for an existing paused session resumes its persisted
configuration. `StopAutoresearch` pauses the loop without deleting `.auto/`.

The returned `Instruction` is the loop prompt. Run it through the normal agent
stream when your host, rather than the CLI command surface, owns each iteration:

```go
run, err := agent.Send(ctx, started.Instruction, nil)
if err != nil {
    return err
}
events, err := run.Stream(ctx)
```

For the slash-command workflow, use `agent.Autoresearch(ctx, objective, nil)`.

## Inspect persisted evidence

```go
status, err := agent.GetAutoresearchStatus(ctx)
history, err := agent.GetAutoresearchHistory(ctx)

for _, attempt := range history.Attempts {
    fmt.Printf("%s replayable=%t pinned=%t state=%s\n",
        attempt.AttemptID,
        attempt.Replayable,
        attempt.Pinned,
        attempt.Materialization,
    )
}
```

Each replayable attempt may contain its latest `AutoresearchEvaluationRecord`
and `AutoresearchDecisionRecord`. Samples retain all configured metrics;
aggregates use median and median absolute deviation so hosts do not need to
reconstruct summaries from console output.

## Replay, rescore, compare, and Pareto

Replay evaluates a candidate in an isolated worktree:

```go
original, err := agent.ReplayAutoresearch(ctx, &autohand.AutoresearchReplayParams{
    AttemptID: candidate.AttemptID,
    Evaluator: autohand.AutoresearchEvaluatorOriginal,
})

current, err := agent.ReplayAutoresearch(ctx, &autohand.AutoresearchReplayParams{
    AttemptID: candidate.AttemptID,
    Evaluator: autohand.AutoresearchEvaluatorCurrent,
})
```

Rescore reuses persisted measurements and appends a decision from the current
policy. The constructors make the mutually exclusive selection explicit:

```go
one, err := agent.RescoreAutoresearch(ctx,
    autohand.AutoresearchRescoreAttempt(candidate.AttemptID))
all, err := agent.RescoreAutoresearch(ctx,
    autohand.AutoresearchRescoreAll())
```

Compare two attempts and inspect the constraint-passing, non-dominated frontier:

```go
comparison, err := agent.CompareAutoresearch(ctx, &autohand.AutoresearchCompareParams{
    LeftAttemptID:  "attempt-1",
    RightAttemptID: "attempt-2",
})
pareto, err := agent.GetAutoresearchPareto(ctx)
```

## Pin and prune safely

Pin artifacts that must survive retention:

```go
_, err := agent.PinAutoresearch(ctx, &autohand.AutoresearchPinParams{
    AttemptID: candidate.AttemptID,
    Pinned:    true,
})
```

Pruning previews by default. Apply only after showing the candidate list and
receiving explicit confirmation:

```go
dryRun := true
preview, err := agent.PruneAutoresearch(ctx, &autohand.AutoresearchPruneParams{
    DryRun: &dryRun,
})

dryRun = false
applied, err := agent.PruneAutoresearch(ctx, &autohand.AutoresearchPruneParams{
    DryRun: &dryRun,
    Yes:    true,
})
```

## Events and hooks

The event stream includes `AutoresearchLifecycleEvent` for start, status, and
pause notifications, plus `AutoresearchOperationEvent` for history, replay,
rescore, compare, Pareto, pin, and prune progress.

```go
switch event := event.(type) {
case autohand.AutoresearchLifecycleEvent:
    fmt.Printf("autoresearch %s: %s\n", event.Phase, event.StatusText)
case autohand.AutoresearchOperationEvent:
    fmt.Printf("autoresearch %s %s success=%t\n",
        event.Operation, event.Phase, event.Success)
}
```

Hook constants cover the full lifecycle from `HookEventAutoresearchStart`
through `HookEventAutoresearchComplete` and `HookEventAutoresearchError`.

See [`examples/27-autoresearch-ledger`](../examples/27-autoresearch-ledger) for
an end-to-end runnable workflow.
