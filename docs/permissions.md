# Permissions

The CLI asks before risky actions such as shell commands and file writes. The Go
SDK surfaces these requests as `PermissionRequestEvent` values and exposes
helpers for responding.

## Permission Modes

Set the mode at startup:

```go
sdk := autohand.NewSDK(&autohand.Config{
    CWD:            ".",
    PermissionMode: autohand.PermissionInteractive,
})
```

| Mode | Behavior |
|---|---|
| `PermissionInteractive` | Emit permission request events. |
| `PermissionUnrestricted` | Allow operations without prompting. |
| `PermissionRestricted` | Ask the CLI to deny risky operations automatically. |
| `PermissionExternal` | Delegate decisions to the host flow when supported by the CLI. |

`PermissionInteractive` is the default. Startup skips the permission-mode RPC
for that default and only sends the RPC for non-default modes.

## Responding To Requests

```go
events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
    Message: "Run the test suite",
})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    if e, ok := event.(autohand.PermissionRequestEvent); ok {
        fmt.Printf("Tool: %s\n", e.Tool)
        fmt.Printf("Description: %s\n", e.Description)

        if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
            log.Printf("allow permission: %v", err)
        }
    }
}
```

## Decision Helpers

```go
err := sdk.AllowPermission(ctx, requestID, autohand.ScopeOnce)
err = sdk.AllowPermission(ctx, requestID, autohand.ScopeSession)
err = sdk.DenyPermission(ctx, requestID, autohand.ScopeOnce)
err = sdk.DenyPermission(ctx, requestID, autohand.ScopeUser)
```

For exact decisions:

```go
err := sdk.PermissionResponse(ctx, requestID, autohand.DecisionAllowAlwaysProject)
```

Available scopes:

- `ScopeOnce`
- `ScopeSession`
- `ScopeProject`
- `ScopeUser`

## Fine-Grained Settings

Use `PermissionSettings` to pass allow and deny preferences into the CLI:

```go
sdk := autohand.NewSDK(&autohand.Config{
    Permissions: &autohand.PermissionSettings{
        Mode:      autohand.PermissionInteractive,
        AllowList: []string{"read_file", "git_status"},
        DenyList:  []string{"delete_path"},
    },
})
```

## Unattended Runs

For scripts that should run without manual prompts, prefer explicit limits:

```go
sdk := autohand.NewSDK(&autohand.Config{
    AutoMode:      true,
    MaxIterations: 8,
    MaxRuntime:    20,
    PermissionMode: autohand.PermissionRestricted,
})
```

Use `Unrestricted` only for trusted workspaces and trusted prompts.
