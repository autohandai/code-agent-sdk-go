# Memory

Autohand CLI memory is exposed through the same agent/tool event stream as other
CLI features. The SDK does not write memory directly; it prompts the agent, and
the agent decides when to call the CLI memory tools.

## Saving Memory

```go
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
    Message: "Remember that this team prefers small Go packages and table tests.",
})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch e := event.(type) {
    case autohand.ToolStartEvent:
        if e.ToolName == string(autohand.ToolSaveMemory) {
            fmt.Println("agent is saving memory")
        }
    case autohand.MessageUpdateEvent:
        fmt.Print(e.Delta)
    }
}
```

## Recalling Memory

Start a later session and ask the agent to check memory:

```go
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
    Message: "What coding preferences should you remember for this repository?",
})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    if e, ok := event.(autohand.MessageUpdateEvent); ok {
        fmt.Print(e.Delta)
    }
}
```

The CLI searches its memory store when the agent calls `recall_memory`.

## Inspecting Context Usage

Memory files count toward context usage:

```go
usage, err := sdk.GetContextUsage(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("memory files: %d\n", usage.MemoryFiles)
```

## Limits

- The agent chooses what to save.
- Recall depends on the agent deciding to use memory.
- Memory content consumes context window budget.
- Memory file paths and persistence behavior are owned by the CLI.
