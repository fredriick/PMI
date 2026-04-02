# Use Cases

## System Administrator

| # | Use Case | Description |
|---|----------|-------------|
| 1 | **Register proxy node** | Register datacenter or residential nodes with metadata (country, city, ISP, IP, OS) via Admin API or dashboard |
| 2 | **Deregister proxy node** | Remove a node from the network when decommissioning or for maintenance |
| 3 | **Monitor network health** | View real-time metrics (requests, success rate, latency, uptime) on the dashboard |
| 4 | **Manage API keys** | Create, list, revoke API keys with optional TTL expiration and SHA-256 hashing |
| 5 | **Set per-key rate limits** | Configure custom rate limits per API key (e.g., 500 req/60s for premium clients) |
| 6 | **Manage IPv6 subnet pools** | Create subnet pools, allocate /64 subnets to datacenter nodes, deallocate when done |
| 7 | **View active sessions** | Monitor and delete active proxy sessions with their associated nodes and TTLs |
| 8 | **Review cooldowns** | View domain-specific node cooldowns that auto-expire via Redis TTL |
| 9 | **Review audit logs** | Query structured audit trail of admin actions filtered by date and action type |
| 10 | **View capacity reports** | Analyze node utilization, bandwidth trends, and receive predictive scaling recommendations |
| 11 | **Configure compliance rules** | Edit blocked domain list (wildcard support) and toggle KYC requirements via `config.yaml` hot reload |
| 12 | **Run load tests** | Use built-in CLI tool to simulate concurrent traffic, measure RPS/latency, validate performance |

## API Consumer / Proxy Client

| # | Use Case | Description |
|---|----------|-------------|
| 13 | **HTTP proxy forwarding** | Route HTTP/HTTPS requests through the gateway using `CONNECT` tunneling |
| 14 | **WebSocket proxy** | Establish WebSocket connections through the gateway with bidirectional tunneling |
| 15 | **Geo-targeted routing** | Target exit nodes by country (`-country-us`) or city (`-city-ny`) via connection string modifiers |
| 16 | **Session persistence** | Maintain the same exit IP across requests using `-session-[id]` modifier |
| 17 | **Authenticate via API key** | Authenticate requests using Bearer token or `Proxy-Authorization` header |
| 18 | **Receive proxy response headers** | Get metadata about the routing: node ID, latency, local address, session cache status |

## Residential Peer Operator

| # | Use Case | Description |
|---|----------|-------------|
| 19 | **Join the mesh network** | Use the Peer SDK to register as a residential exit node via gRPC |
| 20 | **Automatic eligibility checks** | SDK validates battery level, CPU usage, and WiFi type before allowing participation |
| 21 | **Grant/revoke consent** | Explicit user consent management via `ConsentManager` with enable/disable callbacks |
| 22 | **Automatic reconnection** | SDK reconnects with exponential backoff (1s to 5min) on disconnection, re-checking eligibility |
| 23 | **Report system metrics** | SDK reads real battery, CPU, WiFi, IP from the OS (Linux `/sys`, macOS `pmset`, Windows `WMIC`) |
| 24 | **Report bandwidth usage** | Submit bandwidth contribution data to the gateway via gRPC `ReportBandwidth` RPC |
| 25 | **Earn payouts** | Receive compensation calculated per GB sent ($0.50) / received ($0.30) with minimum threshold ($10) |

## Peer Dashboard (PWA)

| # | Use Case | Description |
|---|----------|-------------|
| 37 | **Authenticate via node ID** | Peer operators authenticate with their node ID to receive a session token |
| 38 | **View node dashboard** | Real-time status, battery, CPU, load, country, ISP displayed on a mobile-first dashboard |
| 39 | **Track earnings** | View current month payout amount, rates per GB, and bandwidth breakdown with visual bars |
| 40 | **Manage participation** | Toggle node active/inactive via consent switch, disconnect and clear session |
| 41 | **Install as PWA** | Install the dashboard as a standalone app on mobile or desktop via service worker |
| 42 | **Offline awareness** | Service worker caches static assets and returns offline error for API calls when disconnected |

## Gateway Service (Internal)

| # | Use Case | Description |
|---|----------|-------------|
| 26 | **Intelligent node selection** | Matchmaker selects optimal nodes using weighted score: `reputation / (load + 1)`, filtered by health and cooldown |
| 27 | **Circuit breaker** | Track per-node failures; open circuit after threshold breach, auto-recover on success |
| 28 | **Rate limiting** | Enforce sliding window rate limits (distributed via Redis or local in-memory) per client |
| 29 | **Domain compliance blocking** | Block requests to government (`*.gov`, `*.mil`), financial, and illegal content domains |
| 30 | **Connection pooling** | Reuse TCP connections to peer nodes via per-node pool (max 10 connections) |
| 31 | **Health monitoring** | Background health checks every 30s; mark nodes unhealthy on failure |
| 32 | **mTLS enforcement** | Mutual TLS between clients and gateway for secure communication |
| 33 | **Distributed tracing** | OpenTelemetry tracing for request flow observability |
| 34 | **Prometheus metrics** | Expose `/metrics` endpoint for operational monitoring |
| 35 | **GeoIP auto-detection** | Automatically detect node country from IP address on registration |
| 36 | **Graceful shutdown** | Handle SIGTERM with 15s drain timeout, stop health checks and cooldown cleanup |
