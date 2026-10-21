# CLI compatibility and migration notes

This source revision is unreleased. It targets the paired CLI startup protocol v1 in the matching CLI source change. Set Config.CLIPath to that build for development. Publication must wait for a compatible CLI release; the TypeScript optional platform packages are currently pinned to CLI 0.9.8, which rejects the new startup protocol. A passing test with an explicit development CLI does not prove compatibility with those pinned binaries.

## Ordinary CLI contract

The checked-in [contract snapshot](../contract/current-cli.json) records 69 registered ordinary JSON-RPC operations, 78 notification methods and 17 built-in provider IDs. Convenience controls cover model and permission changes, context usage/compaction, plugins, account information, MCP server management, agent/skill discovery, goals, autoresearch, browser handoff and host step control. CLI slash commands remain available through prompt/command APIs.

The ordinary-session contract does not include restricted Blueprint answer or device-login profiles. Those profiles have their own protocols and admission rules; sending their methods to an ordinary session does not grant access.

## Settings before provider startup

Use Config.RunSettings for native per-run CLI settings. The SDK validates and copies JSON before starting the process. Explicit provider/model/credential fields take precedence over the corresponding raw settings; omitted fields retain saved CLI configuration. Provider credentials are separate from CLI account authentication. Selecting a provider does not replace the signed-in account token.

Overrides apply before provider construction and are not saved back to the user's configuration. Credential values are not put in command-line arguments. JSON false, null, arrays, and literal custom/extension provider IDs are retained where accepted by the CLI schema. Unsafe prototype paths and non-JSON values are rejected.

For example, the following native settings disable a feature and select session awareness:

~~~json
{"features":{"slashGoal":false},"sessions":{"awareness":"warn"}}
~~~

Supported provider IDs are `anthropic`, `autohandai`, `azure`, `bedrock`, `cerebras`, `deepseek`, `llamacpp`, `llmgateway`, `mlx`, `nvidia`, `ollama`, `openai`, `openrouter`, `sakana`, `vertexai`, `xai`, `zai`, plus custom and extension provider IDs understood by the CLI. Availability, credentials, feature gates and local runtime requirements remain the CLI's responsibility. The CLI-backed surface delegates inference to the CLI.

## Sessions and ephemeral runs

Session IDs or paths restore saved history before initialization. Resume/continue selects the latest saved session. Fork requires an explicit source and the CLI's experimental-fork feature; it creates a new session with recorded ancestry. Missing sources and disabled forks fail instead of silently creating an empty session.

Persistence follows the CLI default. Set Config.Ephemeral to true to keep a run out of session history, auto-memory and session synchronization. The legacy persist/persistSession/persist_session fields are retained for source compatibility and do not control automatic persistence. Use the explicit ephemeral option when persistence must be disabled. Ephemeral runs cannot resume or fork saved history; the CLI rejects that combination.

## Complete notifications

Use sdk.OnNotification(callback) to access the original method and JSON parameters: packet.Method and packet.Params; call the returned function to unsubscribe. Typed event streams remain available. Unknown fields, future methods, arrays and null payloads remain accessible. The notification method determines identity; a payload's own type field does not replace the wire method.

Callback observers run synchronously in wire order. Hand off blocking work; do not issue a synchronous RPC from inside a reader callback. Each observer receives an independent payload copy, and ordinary callback failures do not interrupt other observers or pending responses. Remove subscriptions when they are no longer needed.

## Legacy wrapper options

Older wrappers emitted switches that the CLI never implemented. Named options generating the following switches now fail before a child process starts. Public option structures remain present for migration.

| Legacy switches | Supported alternative |
| --- | --- |
| --agents-md, --no-agents-md, --agents-md-create, --agents-md-auto-update, --agents-md-path | Use normal workspace AGENTS.md discovery; --bare disables automatic discovery. |
| --skills, --skill-sources, --install-missing-skills | Install skills with the registered installSkill RPC or use the CLI's normal skill discovery locations. |
| --auto-save-interval | Use the CLI's automatic persistence or explicit ephemeral mode; there is no timer override. |
| --compression-threshold, --summarization-threshold | Use the context-compaction RPC for on/off control; these threshold switches do not exist. |
| --permission-mode, --permission-allow-list, --permission-deny-list | Use runtime permission controls, or supported --allowed-tools/--disallowed-tools through raw CLI arguments. |

Raw extra arguments are forwarded to the CLI parser unchanged. They are outside the generated-option guard, so a paired or custom CLI can interpret its own supported flags. Legacy RPC helpers whose methods are absent from the snapshot's registeredMethods are likewise not ordinary CLI capabilities; the CLI may return method-not-found.

## Verification

From a source checkout, run:

~~~sh
python3 scripts/check_cli_contract.py --cli /path/to/cli-3 --language go
~~~

This checks source-level contract drift and named operation coverage. It is diagnostic evidence, not proof of runtime behavior, provider health, package installation, publication or release compatibility. Run the native tests and actual-CLI fixtures separately. Runtime fixtures must use local mock account/inference services, never real inference credentials.
