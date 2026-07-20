# Changelog

## Unreleased

### Added

- Typed skill registry discovery and installation APIs.
- Typed MCP server, tool, and configuration discovery APIs.
- A deterministic three-metric startup performance gate and documentation.

### Fixed

- Removed the unconditional 500 ms transport startup delay.
- Made SDK startup transactional and verified CLI readiness before committing
  lifecycle state.
- Made transport shutdown bounded and cleared stopped process state.
- Broadcast events independently so subscribers no longer steal one another's
  notifications.
- Removed canceled event subscriptions so they cannot consume later events.
- Failed and drained pending requests immediately when CLI stdout reaches EOF.
- Returned typed lifecycle errors for requests made before transport startup.
