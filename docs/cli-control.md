# Conversation, Browser Handoff, And Auto-Mode Control

The typed methods in this guide are available on `RPCClient`, `SDK`, and
`Agent`.

## Reset A Conversation

Call `Reset(ctx)` to clear the active conversation and start a new session. The
returned `ResetResult.SessionID` is assigned by the CLI.
