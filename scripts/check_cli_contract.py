#!/usr/bin/env python3
"""Compare the checked-in SDK contract with an actual CLI source checkout.

This is a drift diagnostic. It does not replace native tests, installed-package
proof, real CLI execution with mock providers, or release verification.
"""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path


def constants(source: str, name: str) -> dict[str, str]:
    match = re.search(r"export const " + name + r" = \{(.*?)\} as const", source, re.S)
    if match is None:
        raise ValueError(f"Cannot read CLI constant {name}")
    return dict(re.findall(r"([A-Z0-9_]+):\s*'([^']+)'", match.group(1)))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cli", required=True, type=Path)
    parser.add_argument("--language", required=True, choices=["typescript", "python", "go", "rust", "java", "csharp", "ruby", "cpp", "swift"])
    parser.add_argument("--sdk", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    snapshot = json.loads((args.sdk / "contract/current-cli.json").read_text())
    rpc = (args.cli / "src/modes/rpc/types.ts").read_text()
    dispatcher = (args.cli / "src/modes/rpc/index.ts").read_text()
    methods = constants(rpc, "RPC_METHODS")
    names = set(re.findall(r"case RPC_METHODS\.([A-Z0-9_]+)\s*:", dispatcher))
    registered = sorted({methods[name] for name in names})
    adapter = (args.cli / "src/modes/rpc/adapter.ts").read_text()
    emitted_literals = set()
    for match in re.finditer(r"writeNotification\(([^\n]+)", adapter):
        emitted_literals.update(re.findall(r"'(autohand\.[^']+)'", match.group(1)))
    notifications = sorted({*constants(rpc, "RPC_NOTIFICATIONS").values(), *emitted_literals})
    factory = (args.cli / "src/providers/ProviderFactory.ts").read_text()
    providers = sorted(set(re.findall(r"case ['\"]([^'\"]+)['\"]\s*:", factory)))
    failures = []
    for key, current in [("registeredMethods", registered), ("notifications", notifications), ("providers", providers)]:
        if snapshot[key] != current:
            failures.append(f"{key} changed: added={sorted(set(current)-set(snapshot[key]))}, removed={sorted(set(snapshot[key])-set(current))}")
    roots = {
        "typescript": ["src"], "python": ["src/autohand_sdk"], "go": ["."],
        "rust": ["src"], "java": ["src/main/java"], "csharp": ["src"],
        "ruby": ["lib"], "cpp": ["src", "include"], "swift": ["Sources"],
    }
    suffixes = {".ts", ".py", ".go", ".rs", ".java", ".cs", ".rb", ".cpp", ".hpp", ".swift"}
    files = []
    for root in roots[args.language]:
        base = args.sdk / root
        candidates = base.glob("*.go") if args.language == "go" else base.rglob("*")
        files.extend(path for path in candidates if path.is_file() and not path.is_symlink() and path.suffix in suffixes and "workersource" not in path.name.lower().replace("_", "") and "__tests__" not in path.parts and not path.name.endswith("_test.go"))
    source = "\n".join(path.read_text() for path in files)
    missing = [method for method in registered if method not in source]
    if missing:
        failures.append(f"Registered RPC operations missing from SDK source: {missing}")
    missing_guards = [flag for flag in snapshot["unsupportedLegacyFlags"] if flag not in source]
    if missing_guards:
        failures.append(f"Legacy compatibility catalog incomplete: {missing_guards}")
    cli_options = (args.cli / "src/index.ts").read_text()
    missing_startup = [flag for flag in snapshot["privateStartupOptions"] if flag not in cli_options]
    if missing_startup:
        failures.append(f"Paired CLI startup protocol missing: {missing_startup}")
    if failures:
        raise SystemExit("\n".join(failures))
    print(json.dumps({"language": args.language, "registeredMethods": len(registered), "notifications": len(notifications), "diagnosticOnly": True, "result": "pass"}))


if __name__ == "__main__":
    main()
