# Startup Performance

The SDK has a deterministic startup budget for work controlled by this wrapper.
`TestStartupBudgets` uses a local JSON-RPC fixture, five warmup iterations, and
50 measured samples for each metric. It reports the median and p95 and fails if
any p95 reaches 50 ms.

The metrics have deliberately narrow boundaries:

- `publicImportMs`: the package-initialization interval recorded from the
  earliest package variable initializer through package `init`, read from a
  fresh Go test process. Go runtime process boot is outside the timer.
- `sdkStartReturnMs`: elapsed time for the public `SDK.Start` API, including its
  readiness `getState` request.
- `fixtureSpawnToFirstRpcMs`: elapsed time to start the deterministic CLI
  fixture and complete a successful `getState` request through `RPCClient`.

Baseline captured on 2026-07-20:

| Metric | Median | p95 | Budget |
| --- | ---: | ---: | ---: |
| `publicImportMs` | 0.001000 ms | 0.001208 ms | < 50 ms |
| `sdkStartReturnMs` | 6.405417 ms | 7.070250 ms | < 50 ms |
| `fixtureSpawnToFirstRpcMs` | 6.243750 ms | 6.837833 ms | < 50 ms |

Run the benchmark directly:

```bash
go test -run TestStartupBudgets -count=1 -v
```

The test emits one machine-readable JSON object with top-level `language`,
`budgetMs`, `metrics`, and `passed` fields. Each metric contains `samples`,
`medianMs`, `p95Ms`, `maxMs`, and `passed`. The fixed protocol is five warmups
and 50 measured samples.

These measurements isolate wrapper overhead. Startup of a real Autohand CLI,
provider authentication, network access, model loading, and provider readiness
depend on the host environment and are intentionally reported separately from
the deterministic 50 ms gate.
