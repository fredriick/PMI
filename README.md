# ProxyMesh

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
├── admin.go                 # Admin API routes (nodes, cooldowns, subnets)
├── config.yaml              # Configuration file
├── docker-compose.yml       # Local development environment
├── gateway/                 # HTTP proxy gateway
│   ├── service.go           # Request routing, auth, dashboard
│   ├── compliance.go        # KYC & domain blocking (wildcard support)
│   ├── ratelimit.go         # Per-client rate limiting
│   ├── metrics.go           # Prometheus metrics endpoint
│   ├── logger.go            # Structured JSON logging
│   └── tracing.go           # OpenTelemetry tracing
├── matchmaker/              # Node selection service
│   ├── service.go           # Selection logic with circuit breaker
│   └── redis_client.go      # Redis data access with cooldown TTL
├── peer-sdk/                # Residential node SDK
│   └── sdk.go               # Node eligibility & consent management
├── payout/                  # Payout calculation service
│   └── service.go           # Compensation calculation for peers
├── internal/                # Shared packages
│   ├── config/              # Configuration loader
│   ├── models/              # Data models
│   ├── grpc/                # gRPC peer service (proto + server)
│   └── subnet/              # IPv6 subnet allocator
├── web/                     # Admin dashboard (static HTML/JS)
│   └── index.html           # Node management UI
├── cmd/
│   └── loadtest/            # Load testing CLI tool
│       └── main.go
└── docs/                    # Documentation
```

## Configuration

Edit `config.yaml`:

```yaml
gateway:
  host: "0.0.0.0"
  port: 8000
  mtls_enabled: true
  circuit_breaker_threshold: 5
  rate_limit_requests: 100
  rate_limit_window_seconds: 60
  tracing_enabled: true

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
```

## Web Dashboard

Access the admin dashboard at `http://localhost:8000/dashboard`.

Features:
- **Overview** - Real-time metrics (requests, success rate, latency, uptime)
- **Nodes** - View, register, and deregister nodes
- **Cooldowns** - View active domain cooldowns with TTL
- **Subnets** - Manage IPv6 subnet pools and allocations
- Auto-refreshes every 10 seconds

## Load Testing

Build and run the load testing tool:

```bash
go build -o loadtest ./cmd/loadtest/
./loadtest -url http://localhost:8000 -c 20 -d 60s -nodes 10
```

| Flag | Default | Description |
|------|---------|-------------|
| `-url` | `http://localhost:8000` | Gateway base URL |
| `-c` | `10` | Concurrent workers |
| `-d` | `30s` | Test duration |
| `-nodes` | `5` | Synthetic nodes to register |
| `-target` | `https://httpbin.org/get` | Target URL for requests |
| `-country` | `US` | Country for node registration |
| `-key` | `test-admin-key` | Admin API key |

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
| `Disconnect` | Graceful node deregistration |

## Development

```bash
go mod tidy          # Update dependencies
go build .           # Build binary
go run .             # Run directly
go build ./...       # Build all packages
```

## Features

- **HTTP CONNECT Proxy** - Full proxy support with target modifiers
- **mTLS Support** - Mutual TLS between clients and gateway
- **Rate Limiting** - Per-client sliding window rate limiter
- **Metrics** - Prometheus metrics at `/metrics`
- **Structured Logging** - JSON logging with request IDs
- **OpenTelemetry Tracing** - Distributed tracing support
- **gRPC Peer Service** - Residential node communication on port 9000
- **Bandwidth Tracking** - Per-peer bandwidth recording with Redis pipelines
- **Payout Calculation** - Automatic compensation calculation per GB
- **Session Persistence** - Redis-backed session-to-node mapping with TTL
- **Circuit Breaker** - Per-node failure tracking with automatic recovery
- **Health Monitoring** - Background node health checks every 30s
- **Geographic Routing** - Country and city-level node selection
- **IPv6 Subnet Allocation** - Pool-based /64 subnet assignment for datacenter nodes
- **Cooldown Management** - Domain-specific node cooldowns with auto-expiration
- **Admin Dashboard** - Web UI for node/cooldown/subnet management
- **Load Testing** - Built-in synthetic traffic generator

## Requirements

- Go 1.21+
- Redis 7.x
- Docker (optional)
