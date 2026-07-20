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
