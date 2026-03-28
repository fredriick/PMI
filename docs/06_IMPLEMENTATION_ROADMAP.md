# Implementation Roadmap

## Status: Phase 2 Complete ✓

---

## Completed ✓

### Core Services
- [x] **Gateway Service** - HTTP server with auth middleware, target parsing, routing
- [x] **Matchmaker** - Redis-based node selection with sorted sets, cooldown filtering
- [x] **Compliance Service** - KYC validation and blocked domains filtering
- [x] **Peer SDK** - Residential node eligibility checks (battery, CPU, WiFi)

### Infrastructure
- [x] **Configuration** - config.yaml with all service settings
- [x] **Docker Compose** - Local development environment
- [x] **Dockerfiles** - Gateway and Matchmaker containers

### Documentation
- [x] **Infrastructure Blueprint** - 01_INFRASTRUCTURE_BLUEPRINT.md
- [x] **Gateway Implementation** - 02_GATEWAY_IMPLEMENTATION.md
- [x] **Peer SDK Onboarding** - 03_PEER_SDK_ONBOARDING.md
- [x] **API Documentation** - 04_API_DOCUMENTATION.md
- [x] **Compliance & Legal** - 05_COMPLIANCE_AND_LEGAL.md

### Gateway Features
- [x] **HTTP CONNECT Proxy** - Real proxy forwarding via CONNECT method
- [x] **Session Persistence** - `-session-[id]` support for sticky IPs
- [x] **mTLS Support** - Mutual TLS between clients and gateway
- [x] **Rate Limiting** - Per-client request limits

### Matchmaker Features
- [x] **Health Monitoring** - Automatic periodic health checks on nodes
- [x] **Load Balancing** - Consider node load in selection logic
- [x] **Geographic Routing** - City-level selection support

### Observability
- [x] **Metrics Endpoints** - Prometheus metrics for monitoring
- [x] **Logging** - Structured logging with request IDs
- [x] **Tracing** - OpenTelemetry integration

### Peer SDK Features
- [x] **Peer Communication** - gRPC/WebSocket for peer-gateway communication
- [x] **Bandwidth Tracking** - Report bandwidth usage per peer
- [x] **Payout Calculation** - Calculate compensation for residential peers

### Phase 2 Additions
- [x] **Automatic Domain Cooldown Expiration** - Redis TTL on cooldown sets, background cleanup loop, configurable TTL via `cooldown_ttl_minutes`
- [x] **IPv6 Subnet Allocation** - Pool-based /64 subnet management for datacenter nodes, allocate/deallocate via admin API
- [x] **Web UI for Node Management** - Dark-themed admin dashboard at `/dashboard` with overview metrics, node CRUD, cooldown viewer, subnet pool manager
- [x] **Load Testing Tool** - CLI tool in `cmd/loadtest/` with configurable concurrency, duration, node count; reports RPS and latency stats
- [x] **Compliance Wildcard Fix** - Corrected `IsBlocked()` to properly match `*.domain` patterns against subdomains and bare domains
- [x] **List Nodes Handler** - Implemented actual Redis-backed node listing (was a stub)

---

## Pending

### Gateway Features
- [ ] **WebSocket Proxy Support** - Upgrade-aware proxying for WS/WSS connections
- [ ] **Connection Pooling** - Reuse TCP connections to exit nodes
- [ ] **Sticky Session Expiration UI** - Dashboard controls for session TTL

### Matchmaker Features
- [ ] **Weighted Selection** - Factor in reputation score, not just load
- [ ] **GeoIP Database Integration** - Automatic country/city detection from IP
- [ ] **Node Capacity Planning** - Predictive scaling based on traffic patterns

### Peer SDK Features
- [ ] **Real System Metrics** - Actual battery/CPU/WiFi detection (currently stubbed)
- [ ] **Automatic Reconnection** - Exponential backoff on disconnect
- [ ] **Multi-platform Support** - Windows, macOS, Linux native agents

### Infrastructure
- [ ] **Distributed Rate Limiting** - Redis-backed rate limiter for multi-instance deployments
- [ ] **Health Check Endpoint** - `/health` for load balancer probes
- [ ] **Graceful Shutdown** - Drain connections on SIGTERM
- [ ] **Configuration Hot Reload** - Watch config.yaml for changes

### Testing
- [ ] **Unit Tests** - Test coverage for all packages
- [ ] **Integration Tests** - End-to-end proxy flow tests
- [ ] **Compliance Tests** - Verify domain blocking and KYC enforcement

### Security
- [ ] **API Key Storage** - Hashed keys in Redis instead of header comparison
- [ ] **Rate Limit Per API Key** - Separate limits per authenticated user
- [ ] **Audit Logging** - Immutable log of admin actions

---

## Ideas / Backlog

- Node geographic distribution visualization (map-based)
- Residential node mobile app (iOS/Android)
- Prometheus Grafana dashboards (pre-built JSON)
- gRPC streaming for real-time node telemetry
- Multi-region gateway federation
- Bandwidth-based pricing tiers
