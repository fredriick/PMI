# Implementation Roadmap

## Status: Phase 4 Complete ✓

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
- [x] **Unit Tests** - 26 tests across compliance, rate limiter, matchmaker, subnet packages
- [x] **Health Check Endpoint** - `/health` returning status + version for load balancer probes
- [x] **Graceful Shutdown** - SIGINT/SIGTERM signal handling with 15s drain timeout, stops background loops

### Phase 3 Additions
- [x] **Distributed Rate Limiting** - Redis sorted set sliding window (`DistributedRateLimiter`), interface-based with `LocalRateLimiter` fallback, `rate_limit_distributed` config toggle
- [x] **API Key Hashing** - SHA-256 hashed keys stored in Redis, create/list/revoke via `/api/keys`, TTL support, auth middleware validates hashed keys
- [x] **Connection Pooling** - Per-node TCP connection pool (`ConnPool`), configurable max size and dial timeout, `Get/Put/Close/Stats` API
- [x] **Integration Tests** - 16 tests: health endpoint, blocked domains (gov, bank, allowed), auth (missing, basic, query params), metrics, rate limiting, session/host/target extraction
- [x] **Connection Pool Tests** - 5 tests: dial new, put/get reuse, max size enforcement, close cleanup, dial failure
- [x] **52 Total Tests** - All passing across 5 test files

### Phase 4 Additions
- [x] **Configuration Hot Reload** - Viper `WatchConfig` with fsnotify, subscriber callback pattern via `config.OnChange()`, compliance service auto-reloads on config change
- [x] **Per-Key Rate Limiting** - `SetKeyRateLimit`/`GetKeyRateLimit` on APIKeyService, admin endpoint `POST /api/keys/ratelimit`, configurable requests-per-window per key
- [x] **Audit Logging** - `AuditLogger` writes structured JSON to file + Redis lists, `GET /api/admin/audit?date=&action=&limit=`, middleware auto-logs non-GET admin requests

---

## Pending

### Gateway Features
- [ ] **WebSocket Proxy Support** - Upgrade-aware proxying for WS/WSS connections
- [ ] **Sticky Session Expiration UI** - Dashboard controls for session TTL

### Matchmaker Features
- [ ] **Weighted Selection** - Factor in reputation score, not just load
- [ ] **GeoIP Database Integration** - Automatic country/city detection from IP
- [ ] **Node Capacity Planning** - Predictive scaling based on traffic patterns

### Peer SDK Features
- [ ] **Real System Metrics** - Actual battery/CPU/WiFi detection (currently stubbed)
- [ ] **Automatic Reconnection** - Exponential backoff on disconnect
- [ ] **Multi-platform Support** - Windows, macOS, Linux native agents

---

## Ideas / Backlog

- Node geographic distribution visualization (map-based)
- Residential node mobile app (iOS/Android)
- Prometheus Grafana dashboards (pre-built JSON)
- gRPC streaming for real-time node telemetry
- Multi-region gateway federation
- Bandwidth-based pricing tiers
