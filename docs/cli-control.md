# Conversation, Browser Handoff, And Auto-Mode Control

The typed methods in this guide are available on `RPCClient`, `SDK`, and
`Agent`.

## Reset A Conversation

Call `Reset(ctx)` to clear the active conversation and start a new session. The
returned `ResetResult.SessionID` is assigned by the CLI.

## Create A Browser Handoff

Call `CreateBrowserHandoff(ctx, params)` to create a continuation token for the
active session. Optional extension and install routing are preserved, and the
result contains all token, session, workspace, timestamp, and URL fields.

## Attach A Browser Handoff

Call `AttachBrowserHandoff(ctx, params)` with a token. A successful result may
include the restored session ID, workspace root, and message count.

Use `AttachLatestBrowserHandoff(ctx)` when the newest unexpired handoff should
be selected without supplying a token.

## Start Auto-Mode

Call `StartAutomode(ctx, params)` with a required prompt and optional iteration,
completion, worktree, checkpoint, runtime, and cost limits. A successful result
contains the accepted session ID while execution continues in the CLI.

## Inspect Auto-Mode Status

Call `GetAutomodeStatus(ctx)` for the live `Active` and `Paused` flags plus the
optional persisted state, iteration and file counters, branch, and checkpoint.

## Pause Auto-Mode

Call `PauseAutomode(ctx)` to pause the active session. CLI business failures
remain typed results with `Success == false` and an optional `Error`.
