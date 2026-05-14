# lb4a

High-performance, concurrent Layer 7 API Gateway and Load Balancer written in Go. The project prioritizes low overhead, minimal dependencies, and safe hot-reload of JSON configuration for production-grade API routing.

## Purpose and motivation

lb4a is designed as a focused, systems-grade API Gateway that:

- Runs on the Go standard `net/http` stack for predictable performance and low allocations.
- Supports hot-reloading routes without downtime via `SIGHUP`.
- Applies least-connections or round-robin balancing with active health checks.
- Emits structured logs designed for downstream analysis tooling.

## Architecture overview

```
Client
  │
  ▼
Rate Limiter ──► Logger ──► Manual Reverse Proxy ──► Backend
```

The gateway uses a global configuration protected by an `RWMutex`. Configuration changes are applied atomically through `types.SetConfig()` and read via `types.GetConfig()` to avoid data races in high-concurrency environments.

### Request flow (Layer 7 gateway internals)

1. **Route resolution**: Match the request path against configured route prefixes using longest-prefix wins.
2. **Backend selection**: Use least-connections or round-robin, skipping unhealthy backends.
3. **Connection tracking**: For least-connections, increment/decrement backend counters atomically.
4. **Proxying**: Clone the incoming request, forward to the selected backend, and stream the response back to the client without buffering.
5. **Logging**: Emit structured `Access Log` records with status, latency, and client IP.

## Routing and reverse proxying

- Routes are configured via `lb4a.json` as a mapping of path prefixes to one or more backend URLs.
- The gateway performs **longest-prefix match** to prevent route hijacking.
- If no route matches, the gateway forwards the request to the configured `default` backend.
- If all backends for a route are offline, the gateway returns `502 Bad Gateway`.

## Load balancing algorithms

| Algorithm | How it works | Concurrency behavior |
| --- | --- | --- |
| Round-robin | Per-route atomic counter selects the next backend index | Lock-free selection using `atomic.AddUint64` |
| Least-connections | Chooses the backend with the lowest active request count | Counts are incremented/decremented with `atomic` |

`least-connection`, `least-conn`, or `2` in configuration selects least-connections; any other value defaults to round-robin.

## Health checks and failover

- A background health worker checks each backend every 2 seconds.
- Each backend is queried via `GET /health` with a bounded timeout.
- Backends are marked dead/alive using a per-backend `RWMutex`.
- Dead backends are skipped by both load balancers.
- When all backends for a route are dead, the gateway responds with `502`.

## Concurrency, resilience, and fault tolerance

Key thread-safety decisions:

- **Global config** guarded by `sync.RWMutex` through `GetConfig`/`SetConfig`.
- **Backend health state** guarded by per-backend `RWMutex`.
- **Active connections** tracked with atomic counters.
- **Round-robin state** stored in a `sync.Map` with atomic counters per route.
- **Rate limiter** stored in a concurrent `sync.Map` keyed by client IP.

These choices minimize lock contention in the proxy hot path and preserve correctness under load.

## Timeout handling

- **Health checks** use an explicit timeout per probe.
- **Proxy requests** currently use the default `http.Client` (no explicit timeout).
- The JSON config includes `timeOut`, but it is not wired into the proxy yet. See the Future Improvements section for production hardening guidance.

## Logging and monitoring

Structured logging uses `log/slog` with JSON output. Important signatures used by analysis scripts:

- `Access Log` for request traffic (status, latency, client IP).
- `SERVER OFFLINE - Evicting from pool` for health check failures.
- `Hot-reload successful!` for configuration reloads.

The `result.sh` script parses log files and extracts request tables, health transitions, status counts, and P95 latency.

## Configuration

### `lb4a.json` schema

```json
{
  "routes": {
    "/api/users": [
      "http://localhost:3000",
      "http://localhost:3001"
    ],
    "/api/files": "http://localhost:3002"
  },
  "default": "http://localhost:3000",
  "timeOut": 500,
  "algo": "least-connection",
  "ratelimit": {
    "requestPerSecond": 10,
    "burst": 20
  }
}
```

Notes:

- Each route value can be **a single string** or **a list of backend URLs**.
- `algo` selects the load balancing strategy.
- `ratelimit` values are per-IP and applied at the gateway edge.

## Setup and installation

### Prerequisites

- Go 1.25+
- Optional demo backend: Python + MongoDB (see `client2/`)

### Install dependencies

```bash
go mod download
```

### Create configuration

Create `lb4a.json` in the repo root using the schema above.

## Running

### Start demo backends (optional)

The `client2` folder contains a Flask-based backend with user and file APIs.

```bash
python client2/main.py 3000
python client2/main.py 3001
```

### Run the gateway

```bash
go run ./...
```

The gateway listens on `:8080`.

### Hot reload configuration

Reloads are handled by `SIGHUP` and tracked via `gateway.pid`:

```bash
go run main.go reload
```

or:

```bash
kill -HUP "$(cat gateway.pid)"
```

## API route examples

Examples based on the demo backend in `client2/`:

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"user":"tarun","role":"admin","status":"active"}'

curl -X GET "http://localhost:8080/api/users?role=admin"
curl -X GET http://localhost:8080/api/users/tarun

curl -X POST http://localhost:8080/api/files/upload \
  -F "files=@test1.txt" \
  -F "files=@test2.txt"

curl -X GET http://localhost:8080/api/files/test1.txt
```

## Backend registration and monitoring

- Backends are registered in `lb4a.json` under `routes`.
- Each backend is probed via `GET /health`.
- Health state is stored per-backend and checked by the load balancers.
- Adding/removing backends requires updating `lb4a.json` and sending `SIGHUP`.

## Testing workflow

Manual tests:

```bash
./testScript.sh
```

Log analysis:

```bash
./result.sh gateway.log
```

The analysis script expects JSON logs and relies on the `Access Log` signature.

## Benchmarking and performance observations

The repository includes `result.txt`, which is a captured log analysis output. It contains request tables, status distributions, and computed P95 latency derived from gateway logs. This provides a baseline for latency behavior under the demo workload and validates that the logging format is compatible with the analysis pipeline.

## Project structure

```
.
├── client2/           # Demo backend service (Flask + MongoDB)
├── connection/        # Reverse proxy + load-balancing algorithms
├── health/            # Active health checks
├── logger/            # Structured access logging middleware
├── parser/            # JSON configuration parsing
├── rateLimmiter/      # IP-based rate limiter middleware
├── reload/            # Hot-reload logic via SIGHUP
├── types/             # Global config + backend models (thread-safe)
├── main.go            # Gateway bootstrap
├── testScript.sh      # Manual test script
├── result.sh          # Log analysis script
└── result.txt         # Sample log analysis output
```

## Future improvements and production considerations

- Wire `timeOut` into the proxy `http.Client` and enforce per-request deadlines.
- Reuse a shared `http.Client` to reduce allocations and improve connection pooling.
- Add active health check intervals and timeouts to configuration.
- Emit structured metrics (p99, error rates) and expose a `/metrics` endpoint.
- Add TLS termination and upstream TLS verification options.
- Support weighted backends and circuit-breaker style backoff.
- Validate configuration on reload to prevent invalid route shapes.
- Add graceful shutdown with context cancellation and draining.

## License

See [LICENSE](LICENSE).
