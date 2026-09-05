# Resumable step control

`PromptParams.StopWhen` pauses a turn after the CLI persists a completed tool
step. The same `Agent` retains its session and can continue from those results.
Use a CLI supporting `autohand.stepEnd` and `autohand.stepDecision`.

```go
condition, err := autohand.IsStepCount(2)
if err != nil {
    return err
}
result, err := agent.Run(ctx, "Inspect the implementation", &autohand.PromptParams{
    StopWhen: []autohand.StopCondition{condition},
})
if err != nil {
    return err
}
if result.Status == "stopped" {
    result, err = agent.Run(ctx, "Continue from your saved findings", nil)
}
```

`HasToolCall("read_file")` stops after that tool appears in a completed step.
Multiple conditions use OR semantics. Each prompt starts its own step count.
`RunResult.Steps` contains the completed tool calls and results; streams expose
the same data as `StepEndEvent`.

A custom `StopCondition` receives a context and completed steps. It may wait
for external work, but must honor cancellation and treat the steps as read-only.
Returning an error first stops the CLI, then returns the error to `Run.Wait`,
`Agent.Run`, or `SDK.Prompt`. `SDK.StreamPrompt` exposes it as an `ErrorEvent`.

Runs on one SDK are serialized through the CLI's completion acknowledgement.
`Run.Abort` cancels that run, including one queued behind another run. Cancel a
stream's context when abandoning it. Cleanup aborts and drains the active turn;
if the CLI does not acknowledge cleanup within two seconds, the SDK closes it.

`Run.Wait` is repeatable. Results distinguish `completed`, `stopped`, `aborted`,
and `error`. Bounded event subscriptions fail with `ErrEventOverflow` when a
consumer falls behind; control notifications are never silently discarded.

For direct `RPCClient` usage, `Prompt` serializes only `stopWhen: {mode: "host"}`.
The caller must consume `StepEndEvent` and call `StepDecision`. Prefer `SDK` or
`Agent` to evaluate callbacks and manage this lifecycle automatically.

The deterministic subprocess suite runs with `go test -race ./...`. To also
exercise an actual CLI's Autohand AI provider, tool execution, persistence, and
continuation against local HTTP mocks:

```sh
AUTOHAND_TEST_CLI_PATH=/path/to/autohand go test -run TestCurrentHarnessStopWhenWithAutohandAI -v
```
