# Implementation Roadmap

## Status: In Progress

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

---

## In Progress

### Node Management
- [x] **Node Registration API** - Endpoints for proxy nodes to register and heartbeat
- [x] **Node Deregistration** - Graceful removal of nodes from pool
- [x] **Node Metadata Updates** - ISP, battery, OS updates from peers

---

## Pending

### Gateway Features
- [x] **HTTP CONNECT Proxy** - Real proxy forwarding via CONNECT method
- [x] **Session Persistence** - `-session-[id]` support for sticky IPs
- [x] **mTLS Support** - Mutual TLS between clients and gateway
- [x] **Rate Limiting** - Per-client request limits

### Matchmaker Features
- [x] **Health Monitoring** - Automatic periodic health checks on nodes
- [ ] **Load Balancing** - Consider node load in selection logic
- [ ] **Geographic Routing** - City-level selection support

### Observability
- [x] **Metrics Endpoints** - Prometheus metrics for monitoring
- [x] **Logging** - Structured logging with request IDs
- [ ] **Tracing** - OpenTelemetry integration

### Peer SDK Features
- [ ] **Peer Communication** - gRPC/WebSocket for peer-gateway communication
- [ ] **Bandwidth Tracking** - Report bandwidth usage per peer
- [ ] **Payout Calculation** - Calculate compensation for residential peers

---

## Ideas / Backlog

- Web UI for node management
- Node geographic distribution visualization
- Automatic domain cooldown expiration
- Residential node mobile app (iOS/Android)
- IPv6 subnet allocation for datacenter nodes
- Load testing tool with synthetic traffic

---

## Ideas / Backlog