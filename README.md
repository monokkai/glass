<div align="center">

# 🔍 glass

**A flight recorder for your Go services.**

_See what your process is doing right now — no Grafana, no Prometheus, no sidecars._

[![Go Reference](https://img.shields.io/badge/go-reference-blue)](https://pkg.go.dev/github.com/you/glass)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen)](https://goreportcard.com/report/github.com/you/glass)

</div>

---

## The problem

Your service is misbehaving in production. Latency spikes, memory climbs, something is stuck — and you have exactly two bad options.

**Option 1:** Stare at `top` and guess.

**Option 2:** Pull up `pprof`, if you remembered to wire it in ahead of time, then decode a wall of raw stack traces under pressure, at 3am, while the incident channel fills up.

Neither gets you an answer fast. `glass` is the third option.

## What it does

Drop one import into your service. When something looks wrong, attach to the running process from another terminal and get a live, human-readable panel — goroutine counts, stuck call sites, memory pressure, and your most recent errors — updating in real time, right there in your shell.

No metrics pipeline running in the background. No cost until you actually need it.

```go
import _ "github.com/monokkai/glass"
```

That's the entire setup.

```bash
$ glass attach 4821
```

```
┌─ glass · my-api-service · pid 4821 ──────────────────┐
│ Goroutines: 142 (▲ 12 last 10s)  Heap: 340MB / 512MB │
│ ─────────────────────────────────────────────────── │
│ Hot paths (by goroutine count):                      │
│   47  net/http.(*conn).serve                          │
│   23  db.(*Pool).worker                                │
│   ⚠ 8  handlers.processPayment [blocked 40s+]         │
│ ─────────────────────────────────────────────────── │
│ Last errors:                                          │
│   payment gateway timeout                              │
│   └── connection reset by peer      2s ago            │
└────────────────────────────────────────────────────┘
```

## Why not just pprof?

`pprof` gives you the raw data. `glass` gives you the answer.

|                        | `pprof`                            | `glass`                      |
| ---------------------- | ---------------------------------- | ---------------------------- |
| Setup                  | manual HTTP handler, ahead of time | one blank import             |
| Output                 | raw stack dump                     | parsed, ranked, annotated    |
| Stuck goroutines       | you find them yourself             | flagged automatically        |
| Runs in the background | no, but you have to remember it    | zero — attach-on-demand only |
| Time to answer         | minutes, under pressure            | seconds                      |

## Install

```bash
go install github.com/monokkai/glass/cmd/glass@latest
```

## Design principles

- **Zero idle cost.** `glass` does nothing until you attach. No background collection, no allocations in your hot path.
- **No infrastructure.** No collector, no storage, no dashboard to run. The terminal is the dashboard.
- **Signal, not noise.** A raw goroutine dump has hundreds of lines. `glass` surfaces the handful that matter.

## License

MIT — see [LICENSE](LICENSE).

---

<div align="center">

_Built because staring at `top` during an incident is not a debugging strategy._

</div>
