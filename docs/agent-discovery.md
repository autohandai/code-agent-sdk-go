# Agent discovery

After starting the SDK, query the running session's effective subagents:

```go
agents, err := sdk.SupportedAgents(ctx)
if err != nil { return err }
for _, agent := range agents { fmt.Println(agent.Name, agent.Source) }
```

Each entry includes an ID (the agent name), name, description, and tool names.
Model, source, and extension ID, version, and scope are optional metadata.
The registry includes built-in, user, external, generated, inline `--agents`,
and enabled extension agents with the CLI's effective precedence. Agent prompts
and local definition paths are excluded.

This calls `autohand.getSupportedAgents` with empty parameters. It requires a CLI
that implements this method; older versions return method-not-found. Malformed
responses raise an error rather than appearing as an empty registry. An empty
agents array is valid.

After an extension command, wait for the turn's completion before querying the
registry again. A prompt RPC acknowledgement alone does not indicate completion.

`SDK.Prompt` waits for turn completion, including extension registry refresh.
`RPCClient.Prompt` remains the lower-level acknowledgement API. The actual CLI
integration in `harness_agents_test.go` uses isolated files and a local auth mock;
run it with `AUTOHAND_TEST_CLI_PATH` pointing to a compatible CLI.
