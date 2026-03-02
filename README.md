<div align="center">

# api-gateway

A lightweight, production-ready API Gateway written in Go.

Request routing · Rate limiting · JWT authentication · Load balancing · Circuit breaking

</div>

---

## What is this?

`api-gateway` is a standalone reverse proxy that sits in front of your backend services — regardless of what language or framework they are written in. You configure it with a single YAML file and run one binary.

Instead of implementing rate limiting, authentication, and load balancing inside every individual service, you handle it once at the gateway level.
```
Client Request
      │
      ▼
┌─────────────────────┐
│      api-gateway    │
│                     │
│  ✓ JWT auth         │
│  ✓ Rate limiting    │
│  ✓ Load balancing   │
│  ✓ Circuit breaking │
└─────────────────────┘
      │           │
      ▼           ▼
  Service A   Service B
  (Node.js)   (Python)
  :3001       :3002
```

---

## Features

| Feature | Description |
|---|---|
| **Reverse Proxy** | Forwards incoming requests to configured backends |
| **Path Routing** | Route `/users` to one service, `/orders` to another |
| **Load Balancing** | Round-robin distribution across multiple backend instances |
| **Rate Limiting** | Token bucket per client IP — returns `429` when exceeded |
| **Circuit Breaker** | Stops routing to failing backends, retries after cooldown |
| **JWT Auth** | Per-route Bearer token validation — returns `401` if invalid |
| **Metrics** | Request count, error count, avg latency at `/metrics` |
| **Hot Reload** | Reload config with `SIGHUP` — zero downtime, no restart |
| **Graceful Shutdown** | Finishes in-flight requests before exiting |
| **Structured Logging** | JSON logs for every request with route, status, latency |

---

## Quickstart

### Binary
```bash
git clone https://github.com/janmang8225/api-gateway
cd api-gateway
go build -o api-gateway ./cmd/gateway
./api-gateway --config config.yaml
```

### Docker
```bash
docker build -t api-gateway .
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml api-gateway
```

### Docker Compose
```bash
docker compose up
```

---

## Configuration

All configuration lives in a single `config.yaml` file.
```yaml
port: 8080
jwt_secret: your-secret-key

routes:
  - path: /users
    auth: true
    backends:
      - http://localhost:3001

  - path: /orders
    auth: false
    backends:
      - http://localhost:3002
      - http://localhost:3003  # multiple backends = automatic load balancing
```

### Config Reference

| Field | Type | Description |
|---|---|---|
| `port` | int | Port the gateway listens on |
| `jwt_secret` | string | Secret key used to validate JWT tokens |
| `routes[].path` | string | Incoming request path to match |
| `routes[].auth` | bool | Whether this route requires a valid JWT token |
| `routes[].backends` | list | One or more backend URLs to forward to |

---

## CLI Flags
```bash
./api-gateway --config config.yaml   # path to config file (default: config.yaml)
./api-gateway --version              # print version and exit
./api-gateway --help                 # show available flags
```

---

## How Each Feature Works

### Routing
Incoming requests are matched by path and forwarded to the configured backend. If no route matches, the gateway returns `404`.
```
GET /users  →  http://localhost:3001/users
GET /orders →  http://localhost:3002/orders
```

### Load Balancing
If a route has multiple backends, requests are distributed using round-robin. Each request goes to the next backend in order.
```yaml
backends:
  - http://localhost:3001
  - http://localhost:3002
  - http://localhost:3003
```

### Rate Limiting
Each client IP gets a token bucket with a burst of 10 requests. Tokens refill at 5 per second. Requests exceeding the limit receive `429 Too Many Requests`.

### Circuit Breaker
Each backend has its own circuit breaker with three states:
```
Closed (healthy)  →  3 failures  →  Open (blocked)
Open (blocked)    →  10s cooldown →  Half-Open (testing)
Half-Open         →  2 successes →  Closed (recovered)
```

When a backend is open, requests to it immediately return `503 Service Unavailable` instead of waiting for a timeout.

### JWT Authentication
Routes with `auth: true` require a valid Bearer token in the `Authorization` header.
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/users
```

Invalid or missing tokens return `401 Unauthorized`. Tokens are validated using HS256 with the `jwt_secret` from your config.

### Metrics
Hit `/metrics` to see live request statistics:
```bash
curl http://localhost:8080/metrics
```
```
route: /users
  requests:      1024
  errors:        3
  avg_latency:   312.50 µs

route: /orders
  requests:      512
  errors:        0
  avg_latency:   198.20 µs
```

### Hot Config Reload
Update your `config.yaml` and send `SIGHUP` — no restart required:
```bash
kill -SIGHUP $(lsof -ti :8080)
```

New routes and backends take effect immediately. In-flight requests are not affected.

---

## Project Structure
```
api-gateway/
├── cmd/
│   └── gateway/
│       └── main.go              # entry point, wires everything together
├── internal/
│   ├── proxy/
│   │   └── proxy.go             # reverse proxy + timeout handling
│   ├── config/
│   │   └── config.go            # yaml loader + hot reload manager
│   ├── balancer/
│   │   └── balancer.go          # round-robin load balancer
│   ├── breaker/
│   │   └── breaker.go           # circuit breaker state machine
│   ├── metrics/
│   │   └── metrics.go           # request metrics collector
│   ├── logger/
│   │   └── logger.go            # structured json logger
│   └── middleware/
│       ├── auth/
│       │   └── auth.go          # jwt authentication middleware
│       └── ratelimit/
│           └── ratelimit.go     # token bucket rate limiter
├── config.yaml                  # example configuration
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

---

## Timeouts

| Timeout | Default | Description |
|---|---|---|
| Upstream | 5s | Max time to wait for a backend response |
| Read | 5s | Max time to read incoming request |
| Write | 10s | Max time to write response to client |
| Idle | 60s | Max time to keep idle connections open |

---

## Requirements

- Go 1.21+ (for building from source)
- Docker (for container deployment)

---

## License

MIT
