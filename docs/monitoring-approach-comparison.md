# Monitoring Approach — Comparison Table

Reference table for system design choices regarding how to handle concurrent website monitoring at different scales.

| Approach | Use Case | Pros | Cons |
|---|---|---|---|
| **Simple (one goroutine per site)** ⭐ (chosen) | < 20 sites, personal monitoring | Easy to understand, no extra deps, trivial to debug | Doesn't scale to hundreds of sites |
| Connection pooling | 20–100 sites | Reuses TCP connections, lower OS overhead | More complex code, marginal benefit for few sites |
| Rate-limited worker pool | 100+ sites | Controls concurrency, prevents self-DoS | Significantly more complex, needs tuning |
| External queue (Redis, etc.) | 1000+ sites, distributed | Horizontal scaling, fault tolerance | Infrastructure overhead, overkill for personal use |

## Why "Simple" for this project?

With fewer than 20 sites to monitor:
- Each goroutine is lightweight (~2 KB stack) — 20 goroutines use negligible memory
- Go's `net/http` client already pools connections internally per host
- No risk of overwhelming the OS with connections
- Code remains trivially readable and debuggable
- If you ever grow beyond 20 sites, upgrading to a worker pool is a contained refactor in `monitor.go`

## When to upgrade?

| Signal | Action |
|---|---|
| > 20 sites monitored | Consider connection pooling |
| > 100 sites or rate-limiting from targets | Switch to a worker pool with configurable concurrency |
| Multiple machines need to monitor | Move to an external queue / distributed architecture |
