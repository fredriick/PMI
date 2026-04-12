# Implementation Roadmap

## Status: Complete ✓

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

### Phase 5 Additions
- [x] **Weighted Node Selection** - `selectByLoad` replaced with weighted scoring combining reputation and load (`reputation / (load + 1)`), `GetNodeReputation`/`UpdateNodeReputation` on RedisClient
- [x] **WebSocket Proxy Support** - Detects `Upgrade: websocket` header, hijacks HTTP connection, establishes CONNECT tunnel to node, bidirectional byte copying with 32KB buffers
- [x] **Real System Metrics** - Peer SDK reads actual battery from `/sys/class/power_supply/BAT0`, CPU from `/proc/stat` with 500ms sample, IP from `net.InterfaceAddrs`, charging status from sysfs

### Phase 6 Additions
- [x] **GeoIP Auto-Detection** - `GeoIPService` with built-in IP-to-country range table, CSV database loader, automatic country detection on node registration when country field is empty
- [x] **Automatic Reconnection** - Peer SDK `Disconnect()`/`TriggerReconnect()` with exponential backoff (1s to 5min max), reconnection goroutine checks eligibility before reconnecting
- [x] **Session Management** - `GET /api/admin/sessions` and `DELETE /api/admin/sessions/:id` endpoints, `ListSessions`/`DeleteSession` on RedisClient, Sessions tab in web dashboard with TTL display
- [x] **57 Total Tests** - 5 new tests (GeoIP lookup, invalid IP, CSV load failure, GeoIP auto-detect, session validation)

### Phase 7 Additions
- [x] **Node Capacity Planning** - `CapacityPlanner` with `AnalyzeNode`/`GenerateReport`/`GetOverloadedNodes`, tracks bandwidth GB/day, utilization %, status (healthy/warning/critical), `GET /api/admin/capacity`, Capacity tab in dashboard
- [x] **Multi-platform Peer SDK Metrics** - Platform-specific implementations via build tags: Linux (`/sys`, `/proc/stat`), macOS (`pmset`, `top`), Windows (`WMIC`), default fallback for unsupported platforms
- [x] **Dashboard Capacity Tab** - Node capacity report with total/healthy/warning/critical counts, per-node GB/day, GB/week, utilization %, load

### Phase 8 Additions (Final)
- [x] **Predictive Scaling** - `GetScalingRecommendations` analyzes fleet capacity and suggests scale_up/scale_down/prepare_scale actions based on critical/warning node counts and average utilization, with urgency levels and confidence scores
- [x] **Bandwidth Trend Tracking** - `StoreBandwidthSnapshot`/`GetBandwidthSnapshots` stores time-windowed bandwidth data in Redis (capped at 48 entries), `calculateGrowthRate` computes growth % from snapshots
- [x] **62 Total Tests** - 5 new capacity planner tests (growth rate, critical scaling, all healthy, high utilization, snapshot fields)

### Phase 9 Additions - Peer Web PWA
- [x] **Peer API Endpoints** - Token-based auth (`POST /api/peer/auth`), node status, bandwidth, earnings, consent toggle, disconnect — all in `peer_api.go`
- [x] **PWA Shell** - `web/peer/` with `manifest.json`, service worker (`sw.js`) for offline caching, installable on mobile/desktop
- [x] **SPA Dashboard** - Three-tab interface: Dashboard (status, battery, CPU, load, bandwidth), Earnings (payout calculation, rates, bandwidth breakdown), Settings (consent toggle, disconnect)
- [x] **Dark-themed UI** - Mobile-first responsive design matching admin dashboard aesthetic
- [x] **Auto-refresh** - Status/bandwidth/earnings refresh every 15 seconds with toast notifications

---

## All roadmap items complete. See Ideas/Backlog for future work.

---

## Ideas / Backlog

- Residential node mobile app (iOS/Android)
- ~~Node geographic distribution visualization~~ (completed)
- ~~Prometheus Grafana dashboards~~ (completed)
- ~~gRPC streaming for real-time node telemetry~~ (skipped - requires protoc)
- ~~Multi-region gateway federation~~ (completed)
- ~~Bandwidth-based pricing tiers~~ (completed)

---

## Recent Additions (Post-Roadmap)

### CLI Enhancements
- [x] **Cobra-based CLI** - `cmd/cli/` with node, key, status management commands
- [x] **Node Commands** - list, register, deregister, status
- [x] **Key Commands** - list, create, delete, rate-limit
- [x] **Status Commands** - health, metrics, capacity

### Features Implemented This Session
- [x] **Peer Operator PWA** - Web-based dashboard for residential node operators
- [x] **Grafana Integration** - Pre-built dashboards with Prometheus metrics
- [x] **Pricing Tiers** - Basic/Premium/Enterprise bandwidth-based pricing
- [x] **Map Visualization** - Geographic node distribution with country flags
- [x] **Multi-region Federation** - FederationService for regional gateway deployment
- [x] **JWT Authentication** - Token-based auth with role support
- [x] **API Versioning** - v1 prefix, deprecation headers, rate limit tiers
- [x] **WebSocket Admin** - Real-time updates via /ws endpoint
- [x] **Admin RBAC** - Role-based access control with superadmin/admin/operator/viewer
- [x] **RBAC UI** - User management in admin dashboard
- [x] **Request Timeouts** - HTTP server configured with Read/Write/Idle timeouts
- [x] **Redis Cluster Support** - Cluster mode with pool_size and max_retries
- [x] **Prometheus PushGateway** - Metrics export to push gateway at intervals
- [x] **Client IP Rate Limiting** - Per-client rate limits with Redis backend, IP whitelist
- [x] **Graceful Shutdown** - Clean HTTP server and prometheus pusher shutdown
- [x] **Traffic Analytics** - /v1/analytics endpoints for summary/country/node stats
- [x] **Request Deduplication** - SHA256-based key for identical concurrent requests, ref counting
- [x] **Retry Logic** - Exponential backoff with jitter for node failures, configurable retryable errors
- [x] **Request Buffering** - SlowClientBuffer with body size limits, RequestBuffer for stats
- [x] **Connection Draining** - ConnectionDrainer with active connection tracking, GracefulShutdown manager
- [x] **Circuit Breaker Dashboard** - API endpoint for circuit breaker status, reset handler
- [x] **Traffic Shaping** - Token bucket rate limiting per client, configurable rates
- [x] **DDoS Protection** - Automatic IP blocking based on failed request threshold
- [x] **Node Latency Ranking** - LatencyTracker with avg/min/max stats, API endpoint
- [x] **Request Queuing** - RequestQueue for overloaded nodes with enqueue/dequeue
- [x] **Reputation Decay** - Automatic reputation decay over time with configurable rate
- [x] **WebSocket Heartbeat** - Track peer connections with ping/pong, alive status
- [x] **Request Validation** - RequestValidator with body size limits, content-type checking
- [x] **Node Selection Strategies** - Random/LeastLoad/LowestLatency/HighestRep/RoundRobin
- [x] **API Request Batching** - BatchProcessor for bulk operations in single request
- [x] **Node Health Score** - Composite score from latency/load/success/reputation/uptime
- [x] **Request Priority** - KeyPriority with weighted priority per API key
- [x] **API Response Caching** - ResponseCache with Redis backend, X-Cache headers
- [x] **Node Retry Handler** - Retry logic with fallback nodes
- [x] **Rate Limit Headers** - X-RateLimit-Limit/Remaining/Reset headers
- [x] **Request ID Generation** - X-Request-ID header with correlation tracking
- [x] **Admin API Metrics** - GET /api/admin/metrics with calls/errors/latencies
- [x] **Node Timeouts** - Configurable per-node dial/read/write/idle timeouts
- [x] **Session Persistence** - Session persistence with failover node support
- [x] **Key Expiration** - API key expiration tracking with renewal
- [x] **Bandwidth Quotas** - Monthly quota management per node
- [x] **Node Groups** - Organize nodes by groups/tags for selection
- [x] **Cost Estimation** - Calculate request costs per client
- [x] **Audit Retention** - Configurable retention policies per audit action
