# Changelog

## Unreleased

### Added

- Resumable per-step stop conditions, typed completed steps, and stopped run results.
- Local HTTP mock integration covering the actual CLI's Autohand AI provider and persisted-step continuation.
- Typed skill registry discovery and installation APIs.
- Typed MCP server, tool, and configuration discovery APIs.
- A deterministic three-metric startup performance gate and documentation.

### Fixed

- Serialize prompt, context, selection, and image fields using the CLI's JSON names.
- Finish runs on `turnEnd`, serialize concurrent prompts, and drain cancelled turns before reuse.
- Make run waits repeatable and keep queued-run cancellation scoped to its own run.
- Surface startup JSON-RPC errors and event-buffer overflow instead of losing them.
- Removed the unconditional 500 ms transport startup delay.
- Made SDK startup transactional and verified CLI readiness before committing
  lifecycle state.
- Made transport shutdown bounded and cleared stopped process state.
- Broadcast events independently so subscribers no longer steal one another's
  notifications.
- Removed canceled event subscriptions so they cannot consume later events.
- Failed and drained pending requests immediately when CLI stdout reaches EOF.
- Returned typed lifecycle errors for requests made before transport startup.
