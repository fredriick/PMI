# ProxyMesh

![ProxyMesh Banner](web/banner.jpg)

A hybrid mesh proxy network combining datacenter and residential exit nodes.

## Architecture

- **Gateway** - Routes user requests to the best exit node based on targeting criteria
- **Matchmaker** - Selects optimal nodes using Redis-backed reputation scoring
- **Peer SDK** - Residential node client with eligibility checks and consent management
- **Payout** - Bandwidth-based compensation calculation for residential peers
- **Subnet Allocator** - IPv6 /64 subnet pool management for datacenter nodes

## Project Structure

```
ProxyMeshProject/
├── main.go                  # Application entry point
├── admin.go                 # Admin API routes (nodes, cooldowns, subnets, keys)
├── peer_api.go              # Peer dashboard API routes (auth, status, earnings)
├── config.yaml              # Configuration file
├── docker-compose.yml       # Local development environment
├── k8s/                     # Kubernetes deployment manifests
├── gateway/                 # HTTP proxy gateway
│   ├── service.go           # Request routing, auth, dashboard
│   ├── compliance.go        # KYC & domain blocking (wildcard support)
│   ├── ratelimit.go         # Rate limiter interface + local implementation
│   ├── redis_ratelimit.go   # Redis-backed distributed rate limiter
│   ├── apikey.go            # SHA-256 hashed API key service
│   ├── connpool.go          # TCP connection pool for exit nodes
│   ├── audit.go             # Audit logging for admin actions
│   ├── geoip.go             # GeoIP country detection from IP ranges
│   ├── metrics.go           # Prometheus metrics endpoint
│   ├── logger.go            # Structured JSON logging
│   ├── tracing.go           # OpenTelemetry tracing
│   ├── compliance_test.go   # Compliance unit tests
│   ├── ratelimit_test.go    # Rate limiter unit tests
│   ├── connpool_test.go     # Connection pool tests
│   └── integration_test.go  # Integration tests
├── matchmaker/              # Node selection service
│   ├── service.go           # Selection logic with circuit breaker
│   ├── redis_client.go      # Redis data access with cooldown TTL
│   └── service_test.go      # Matchmaker unit tests
├── peer-sdk/                # Residential node SDK
│   ├── sdk.go               # Node eligibility & consent management
│   ├── metrics_linux.go     # Linux system metrics (battery, CPU, WiFi)
│   ├── metrics_darwin.go    # macOS system metrics (pmset, top)
│   ├── metrics_windows.go   # Windows system metrics (WMIC)
│   ├── metrics_default.go   # Fallback for unsupported platforms
│   └── metrics_common.go    # Cross-platform (IP, system info)
├── payout/                  # Payout calculation service
│   └── service.go           # Compensation calculation for peers
├── internal/                # Shared packages
│   ├── config/              # Configuration loader
│   ├── models/              # Data models
│   ├── grpc/                # gRPC peer service (proto + server)
│   └── subnet/              # IPv6 subnet allocator
│       ├── allocator.go
│       └── allocator_test.go
├── web/                     # Static web assets
│   ├── index.html           # Landing page
│   ├── docs.html            # Production documentation
│   └── dashboard/
│   │   └── index.html       # Admin dashboard UI
│   └── peer/                # Peer operator PWA
│       ├── index.html       # SPA shell
│       ├── style.css        # Dark-themed styles
│       ├── app.js           # SPA logic
│       ├── manifest.json    # PWA manifest
│       ├── sw.js            # Service worker
│       ├── icon-192.png     # PWA icon
│       └── icon-512.png     # PWA icon
├── cmd/
│   ├── cli/                   # Cobra CLI tool (node, key, status)
│   │   ├── main.go
│   │   ├── node.go
│   │   ├── key.go
│   │   └── status.go
│   └── loadtest/              # Load testing CLI tool
│       └── main.go
└── docs/                    # Documentation
```

## Configuration

Edit `config.yaml`:

```yaml
gateway:
  host: "0.0.0.0"
  port: 8000
  mtls_enabled: false
  circuit_breaker_threshold: 5
  rate_limit_requests: 100
  rate_limit_window_seconds: 60
  rate_limit_distributed: true
  tracing_enabled: true
  request_id_prefix: "req"
  request_id_format: "unix"

matchmaker:
  host: "localhost"
  port: 6379
  pool_size: 10
  cooldown_ttl_minutes: 15

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

compliance:
  blocked_domains:
    - "*.gov"
    - "*.mil"
    - "*.bankofamerica.com"
    - "*.chase.com"
    - "*.wellsfargo.com"
    - "*.illegal-content.*"
  kyc_required: true

peer:
  min_battery_percent: 20
  max_cpu_percent: 80
  require_unmetered_wifi: true

subnet:
  enabled: true
  prefix: "2001:db8::"
  prefix_len: 48
```

## Running Locally

### Using Docker Compose

```bash
docker-compose up -d
```

This starts Redis and both services.

### Manual Setup

1. Start Redis:
   ```bash
   redis-server
   ```

2. Build and run:
   ```bash
   go build -o gateway.exe .
   ./gateway.exe
   ```

3. Run tests:
   ```bash
   go test ./... -v
   ```

### Kubernetes Deployment

ProxyMesh includes Kubernetes manifests for single-region deployments:

```bash
kubectl apply -k k8s/
```

The manifests deploy:

- Redis with persistent storage
- Gateway with health/readiness probes
- LoadBalancer service on ports 80 and 9000
- Optional NGINX ingress for `proxymesh.local`

To build and push your own image:

```bash
docker build -t ghcr.io/fredriick/proxymesh:latest -f Dockerfile.gateway .
kubectl set image deployment/proxymesh-gateway -n proxymesh gateway=ghcr.io/fredriick/proxymesh:latest
kubectl rollout restart deployment/proxymesh-gateway -n proxymesh
```

### Setup Your First Node

See [`docs/09_SETUP_FIRST_NODE.md`](docs/09_SETUP_FIRST_NODE.md) for a step-by-step guide covering Redis setup, node registration, Peer Dashboard authentication, and Raspberry Pi deployment notes.

### Quick Start (Peer PWA)

1. Start Redis and gateway (see above)

2. Register a test node (PowerShell):
   ```powershell
   $body = @{node_id="my-node-01";node_type="residential";country="US";city="New York";isp="Verizon";ip="72.1.2.3";os="linux"} | ConvertTo-Json
   Invoke-RestMethod -Uri "http://localhost:8000/api/admin/nodes" -Method POST -Headers @{"X-Admin-Key"="test-admin-key"} -ContentType "application/json" -Body $body
   ```

3. Open http://localhost:8000/peer/ and enter `my-node-01` as the Node ID

## Admin Dashboard

Open http://localhost:8000/dashboard for the dark-themed admin UI:
- **Nodes** - Real-time node list with country, ISP, reputation, load, last seen
- **Actions** - Reset circuit breaker, eject node, node management
- **Cooldowns** - View active domain cooldowns with TTL
- **API** - JSON endpoints at `/web/api/nodes`, `/web/api/cooldowns`

## API Usage

### Connection Format

```
http://[user]-[targets]:[pass]@gateway.io:8000
```

### Target Modifiers

| Modifier | Example | Description |
|----------|---------|-------------|
| `-country-[code]` | `-country-us` | Filter by country |
| `-city-[name]` | `-city-ny` | Filter by city |
| `-session-[id]` | `-session-abc123` | Keep same IP |

### Response Headers

- `X-Request-ID` - Correlation ID generated from the configured prefix and format, or echoed from the incoming request.
- `X-Proxy-Node-ID` - Exit node identifier
- `X-Proxy-Latency` - Node response time (ms)
- `X-Proxy-Local-Addr` - Bound local address
- `X-Session-Cached` - Set when session was reused

### Example

```bash
curl -x http://user-country-us:pass@gateway.io:8000 https://example.com
```

## Admin API

All admin endpoints require the `X-Admin-Key` header.

### Node Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/admin/nodes` | Register a node |
| GET | `/api/admin/nodes` | List all nodes |
| GET | `/api/admin/nodes/:id` | Get node status |
| POST | `/api/admin/nodes/:id/heartbeat` | Send heartbeat |
| DELETE | `/api/admin/nodes/:id` | Deregister a node |
| GET | `/api/admin/cooldowns` | List active cooldowns |
| GET | `/api/admin/sessions` | List active sessions |
| DELETE | `/api/admin/sessions/:id` | Delete a session |
| GET | `/api/admin/capacity` | Node capacity report |
| GET | `/api/admin/scaling` | Scaling recommendations |
| GET | `/api/admin/audit` | Query audit log entries |
| GET | `/api/admin/request-id` | Get request ID generation settings |
| POST | `/api/admin/request-id` | Update request ID prefix or format |

### API Key Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/keys` | Create a new API key |
| GET | `/api/keys` | List all API keys (hashed) |
| DELETE | `/api/keys` | Revoke an API key |
| POST | `/api/keys/ratelimit` | Set per-key rate limit |

```bash
# Create a key
curl -X POST http://localhost:8000/api/keys \
  -H "X-Admin-Key: your-admin-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "production", "ttl_days": 90}'

# Set per-key rate limit (500 requests per 60 seconds)
curl -X POST http://localhost:8000/api/keys/ratelimit \
  -H "X-Admin-Key: your-admin-key" \
  -H "Content-Type: application/json" \
  -d '{"key": "the-api-key", "requests": 500, "window_seconds": 60}'

# Query audit log for today
curl http://localhost:8000/api/admin/audit?date=2026-03-29&limit=50 \
  -H "X-Admin-Key: your-admin-key"
```

### Subnet Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/subnets/pools` | Create a subnet pool |
| GET | `/api/subnets/pools` | List all pools with allocations |
| POST | `/api/subnets/allocate` | Allocate a subnet to a node |
| DELETE | `/api/subnets/allocate/:nodeID` | Deallocate a subnet |
| GET | `/api/subnets/nodes/:nodeID` | Get a node's subnet |

### Register a Node

```bash
curl -X POST http://localhost:8000/api/admin/nodes \
  -H "X-Admin-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "dc-node-001",
    "node_type": "datacenter",
    "country": "US",
    "city": "New York",
    "isp": "AWS",
    "ip": "54.1.2.3",
    "os": "linux"
  }'

# Peer authentication example
curl -X POST http://localhost:8000/api/peer/auth \
  -H "Content-Type: application/json" \
  -d '{"node_id": "my-residential-node"}'
```

## Web Dashboard

Access the admin dashboard at `http://localhost:8000/dashboard`.

Features:
- **Overview** - Real-time metrics (requests, success rate, latency, uptime)
- **Nodes** - View, register, and deregister nodes
- **Cooldowns** - View active domain cooldowns with TTL
- **Subnets** - Manage IPv6 subnet pools and allocations
- **Sessions** - View/delete active sessions with TTL
- **Capacity** - Node capacity report with utilization and status
- Auto-refreshes every 10 seconds

## Peer Dashboard (PWA)

Access the peer dashboard at `http://localhost:8000/peer/`.

Residential node operators can authenticate with their node ID to view status, track earnings, and manage participation.

### Peer API

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/peer/auth` | None | Authenticate with `node_id`, returns session token |
| GET | `/api/peer/status` | `X-Peer-Token` | Get node status, battery, CPU, load |
| GET | `/api/peer/bandwidth` | `X-Peer-Token` | Get bandwidth data and history |
| GET | `/api/peer/earnings` | `X-Peer-Token` | Get payout calculation and rates |
| POST | `/api/peer/consent` | `X-Peer-Token` | Toggle node active/inactive |
| POST | `/api/peer/disconnect` | `X-Peer-Token` | Disconnect and revoke session |

```bash
# Authenticate
curl -X POST http://localhost:8000/api/peer/auth \
  -H "Content-Type: application/json" \
  -d '{"node_id": "my-residential-node"}'

# Check status (use returned token)
curl http://localhost:8000/api/peer/status \
  -H "X-Peer-Token: <token>"
```

## Load Testing

Build and run the built-in load testing tool to benchmark gateway throughput and latency.

```bash
go build -o loadtest ./cmd/loadtest/
./loadtest -url http://localhost:8000 -c 20 -d 60s -nodes 10
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-url` | `http://localhost:8000` | Gateway base URL |
| `-c` | `10` | Concurrent workers (simulated users) |
| `-d` | `30s` | Total test duration |
| `-nodes` | `5` | Synthetic datacenter nodes to register |
| `-target` | `https://httpbin.org/get` | Target URL for proxy requests |
| `-country` | `US` | Country filter for node registration |
| `-key` | `test-admin-key` | Admin API key for node registration |

### How It Works

1. Registers `N` synthetic datacenter nodes via the admin API
2. Spawns `C` concurrent goroutines to fire proxy requests for `D` duration
3. Each goroutine sends HTTP GET requests via Basic Auth through the gateway
4. Tracks total requests, success/failures, rate-limits (429s), min/max/avg latency
5. Deregisters all synthetic nodes when done

### Example Output

```
ProxyMesh Load Test
===================
Target:     http://localhost:8000
Concurrency: 50
Duration:   30s
Nodes:      10

[1/4] Registering 10 synthetic nodes...
      Registered 10 nodes

[2/4] Starting load test...

[3/4] Results:
  Total Requests:   15000
  Successful:       14980 (99.9%)
  Failed:           5
  Rate Limited:     15
  Requests/sec:     500.2
  Avg Latency:      12.3 ms
  Min Latency:      2 ms
  Max Latency:      89 ms

[4/4] Cleaning up nodes...
      Done.
```

The tool registers synthetic datacenter nodes, fires concurrent proxy requests, reports RPS/latency stats, and cleans up on exit.

## Compliance

The gateway enforces:

- **KYC** - Business customers must verify identity
- **Blocked Domains** - Government, military, financial domains blocked (wildcard `*.domain` matching)
- **Cooldown Expiration** - Domain cooldowns auto-expire via Redis TTL (configurable, default 15 min)
- **Traffic Filtering** - Illegal content prohibited

## gRPC Peer Service

Residential nodes connect via gRPC on port 9000:

| RPC | Description |
|-----|-------------|
| `Connect` | Register node with eligibility checks |
| `Heartbeat` | Report battery, CPU, charging status |
| `ReportBandwidth` | Submit bandwidth usage data |
| `StreamTelemetry` | Bidirectional real-time telemetry stream |
| `Disconnect` | Graceful node deregistration |

## Testing

```bash
go test ./... -v              # Run all tests with output
go test ./gateway/... -v      # Run gateway tests only
go test ./matchmaker/... -v   # Run matchmaker tests only
go test ./... -cover          # Run with coverage
```

135+ tests across 15 test files covering compliance, rate limiting, connection pooling, subnet allocation, matchmaker circuit breaker, GeoIP, capacity planning, integration flows, RBAC, JWT, federation, peer SDK, health, payout, and Web UI.

## Development

```bash
go mod tidy          # Update dependencies
go build .           # Build binary
go run .             # Run directly
go build ./...       # Build all packages
```

## CI/CD

A GitHub Actions workflow runs on every push/PR to `main`:

- **Test matrix** — Go 1.21, 1.22, 1.23 on Ubuntu with a Redis service container
- **Steps** — `go build`, `go vet`, `go test -race -cover`
- **Artifacts** — Coverage reports uploaded per Go version
- **Lint** — `golangci-lint` (non-blocking)

## Documentation

- [`docs/08_DATACENTER_VALUE.md`](docs/08_DATACENTER_VALUE.md) - Datacenter value proposition and deployment model guide
- [`docs/09_SETUP_FIRST_NODE.md`](docs/09_SETUP_FIRST_NODE.md) - Step-by-step setup guide for registering and testing your first node
- [`docs/10_KUBERNETES_DEPLOYMENT.md`](docs/10_KUBERNETES_DEPLOYMENT.md) - Kubernetes deployment guide
- [`docs/11_SECRET_ROTATION.md`](docs/11_SECRET_ROTATION.md) - Production secret and certificate rotation policy

## Features

- **HTTP CONNECT Proxy** - Full proxy support with target modifiers
- **WebSocket Proxy** - Upgrade-aware proxying with bidirectional tunneling
- **mTLS Support** - Mutual TLS between clients and gateway
- **Rate Limiting** - Distributed (Redis) or local (in-memory) per-client sliding window
- **Per-Key Rate Limiting** - Configurable rate limits per API key
- **API Key Auth** - SHA-256 hashed keys in Redis with TTL and revocation
- **Connection Pooling** - Per-node TCP connection pool with configurable max size
- **Config Hot Reload** - Automatic config.yaml reload on file changes (fsnotify)
- **Audit Logging** - Structured audit trail of admin actions to file and Redis
- **Metrics** - Prometheus metrics at `/metrics`
- **Structured Logging** - JSON logging with request IDs
- **Request ID Customization** - Configurable `X-Request-ID` prefix and format (`unix`, `timestamp`, `uuid`)
- **OpenTelemetry Tracing** - Distributed tracing support
- **gRPC Peer Service** - Residential node communication on port 9000
- **Kubernetes Deployment** - Ready-to-apply manifests for Redis, Gateway, LoadBalancer, and optional Ingress
- **Bandwidth Tracking** - Per-peer bandwidth recording with Redis pipelines
- **Payout Calculation** - Automatic compensation calculation per GB
- **Session Persistence** - Redis-backed session-to-node mapping with TTL
- **Circuit Breaker** - Per-node failure tracking with automatic recovery
- **Health Monitoring** - Background node health checks every 30s
- **Geographic Routing** - Country and city-level node selection
- **GeoIP Auto-Detection** - Automatically detects node country from IP on registration
- **Weighted Node Selection** - Combines reputation score and load for optimal routing
- **IPv6 Subnet Allocation** - Pool-based /64 subnet assignment for datacenter nodes
- **Cooldown Management** - Domain-specific node cooldowns with auto-expiration
- **Admin Dashboard** - Web UI for node/cooldown/subnet management
- **Load Testing** - Built-in synthetic traffic generator
- **Health Endpoint** - `/health` for load balancer probes
- **Graceful Shutdown** - SIGTERM handling with 15s drain timeout
- **Real System Metrics** - Peer SDK reads actual battery, CPU, WiFi, IP from system
- **Multi-platform Support** - Linux (/sys, /proc), macOS (pmset, top), Windows (WMIC)
- **Node Capacity Planning** - Bandwidth trend analysis, utilization reporting, capacity alerts
- **Predictive Scaling** - Automatic scaling recommendations based on traffic growth rates

### Peer Dashboard (PWA)
- **Peer Web Dashboard** - Installable PWA at `/peer` for residential node operators
- **Real-time Node Status** - Battery, CPU, load, country, ISP displayed with 15s auto-refresh
- **Earnings Tracker** - Current month payout, rates per GB, bandwidth breakdown
- **Consent Management** - Toggle node active/inactive, disconnect from the network
- **Offline Support** - Service worker caches static assets for offline awareness

## Requirements

- Go 1.21+
- Redis 7.x
- Docker (optional)
