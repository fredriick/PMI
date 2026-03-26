# ProxyMesh

A hybrid mesh proxy network combining datacenter and residential exit nodes.

## Architecture

- **Gateway** - Routes user requests to the best exit node based on targeting criteria
- **Matchmaker** - Selects optimal nodes using Redis-backed reputation scoring
- **Peer SDK** - Residential node client with eligibility checks and consent management

## Project Structure

```
ProxyMeshProject/
├── main.go                  # Application entry point
├── config.yaml              # Configuration file
├── docker-compose.yml       # Local development environment
├── gateway/                 # HTTP proxy gateway
│   ├── service.go           # Request routing & auth
│   ├── compliance.go        # KYC & domain blocking
│   ├── ratelimit.go        # Per-client rate limiting
│   ├── metrics.go          # Prometheus metrics endpoint
│   ├── logger.go           # Structured JSON logging
│   └── tracing.go          # OpenTelemetry tracing
├── matchmaker/              # Node selection service
│   ├── service.go           # Selection logic with circuit breaker
│   └── redis_client.go      # Redis data access
├── peer-sdk/                # Residential node SDK
│   └── sdk.go               # Node eligibility & consent
├── payout/                  # Payout calculation service
│   └── service.go           # Compensation calculation for peers
├── internal/               # Shared packages
│   ├── config/              # Configuration loader
│   ├── models/              # Data models
│   └── grpc/                # gRPC peer service
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

redis:
  host: "localhost"
  port: 6379

compliance:
  blocked_domains:
    - "*.gov"
    - "*.mil"
    - "*.bankofamerica.com"
  kyc_required: true

peer:
  min_battery_percent: 20
  max_cpu_percent: 80
  require_unmetered_wifi: true
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

### Example

```bash
curl -x http://user-country-us:pass@gateway.io:8000 https://example.com
```

## Compliance

The gateway enforces:

- **KYC** - Business customers must verify identity
- **Blocked Domains** - Government, military, financial domains blocked
- **Traffic Filtering** - Illegal content prohibited

## Development

```bash
go mod tidy          # Update dependencies
go build .           # Build binary
go run .             # Run directly
```

## Features

- **HTTP CONNECT Proxy** - Full proxy support with target modifiers
- **mTLS Support** - Mutual TLS between clients and gateway
- **Rate Limiting** - Per-client sliding window rate limiter
- **Metrics** - Prometheus metrics at `/metrics`
- **Structured Logging** - JSON logging with request IDs
- **OpenTelemetry Tracing** - Distributed tracing support
- **gRPC Peer Service** - Residential node communication on port 9000
- **Bandwidth Tracking** - Per-peer bandwidth recording
- **Payout Calculation** - Automatic compensation calculation

## Requirements

- Go 1.21+
- Redis 7.x
- Docker (optional)