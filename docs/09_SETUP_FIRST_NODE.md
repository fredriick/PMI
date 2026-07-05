# Setup Your First ProxyMesh Node

This guide walks through registering a residential node, opening the Peer Dashboard, and verifying that the node can authenticate and report status.

## 1. Start Redis

ProxyMesh stores node state, cooldowns, rate-limit counters, and bandwidth records in Redis.

### Docker Compose

```bash
docker-compose up -d redis
```

### Windows manual install

```powershell
cd F:\PMI\ProxyMeshProject
Start-Process -FilePath "redis\redis-server.exe" -ArgumentList "redis\redis.windows.conf" -WindowStyle Hidden
redis\redis-cli.exe ping
```

Expected output:

```text
PONG
```

## 2. Start the Gateway

### Docker Compose

```bash
docker-compose up -d gateway
```

### Manual Windows build

```powershell
cd F:\PMI\ProxyMeshProject
go build -o proxymesh.exe .
Start-Process -FilePath ".\proxymesh.exe" -WindowStyle Hidden
```

### Manual Linux/macOS build

```bash
go build -o proxymesh .
./proxymesh
```

The gateway listens on:

- Admin dashboard: `http://localhost:8000/dashboard`
- Peer dashboard: `http://localhost:8000/peer/`
- Health endpoint: `http://localhost:8000/health`
- gRPC peer service: `localhost:9000`

## 3. Register a Residential Node

Use the Admin API to register a node before opening the Peer Dashboard.

### PowerShell

```powershell
$body = @{
  node_id = "raspberry-pi-01"
  node_type = "residential"
  country = "US"
  city = "New York"
  isp = "Comcast"
  ip = "72.1.2.3"
  os = "linux"
} | ConvertTo-Json

Invoke-RestMethod `
  -Uri "http://localhost:8000/api/admin/nodes" `
  -Method POST `
  -Headers @{ "X-Admin-Key" = "test-admin-key" } `
  -ContentType "application/json" `
  -Body $body
```

Expected response:

```json
{
  "status": "success",
  "node_id": "raspberry-pi-01",
  "message": "Node registered successfully"
}
```

### Linux/macOS

```bash
curl -X POST http://localhost:8000/api/admin/nodes \
  -H "X-Admin-Key: test-admin-key" \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "raspberry-pi-01",
    "node_type": "residential",
    "country": "US",
    "city": "New York",
    "isp": "Comcast",
    "ip": "72.1.2.3",
    "os": "linux"
  }'
```

## 4. Open the Peer Dashboard

Open:

```text
http://localhost:8000/peer/
```

Enter the node ID:

```text
raspberry-pi-01
```

The dashboard should authenticate and display:

- Node status
- Battery level
- CPU usage
- Load
- Bandwidth sent/received
- Current earnings
- Consent toggle

## 5. Verify Peer API Manually

Authenticate the node:

```bash
curl -X POST http://localhost:8000/api/peer/auth \
  -H "Content-Type: application/json" \
  -d '{"node_id":"raspberry-pi-01"}'
```

Save the returned token and use it for the remaining Peer API calls.

### Status

```bash
curl http://localhost:8000/api/peer/status \
  -H "X-Peer-Token: <token>"
```

### Bandwidth

```bash
curl http://localhost:8000/api/peer/bandwidth \
  -H "X-Peer-Token: <token>"
```

### Earnings

```bash
curl http://localhost:8000/api/peer/earnings \
  -H "X-Peer-Token: <token>"
```

## 6. Verify Admin Dashboard

Open:

```text
http://localhost:8000/dashboard
```

Confirm that `raspberry-pi-01` appears in the node list.

You can also query it through the Admin API:

```bash
curl http://localhost:8000/api/admin/nodes \
  -H "X-Admin-Key: test-admin-key"
```

## 7. Raspberry Pi Notes

For a real Raspberry Pi node:

1. Install Go or run the compiled `peer-sdk` binary.
2. Use a stable node ID such as the Pi hostname.
3. Keep the device on unmetered WiFi.
4. Monitor battery/CPU thresholds in `config.yaml`:

```yaml
peer:
  min_battery_percent: 20
  max_cpu_percent: 80
  require_unmetered_wifi: true
```

5. Use the Peer Dashboard to toggle consent on/off.

## Troubleshooting

### `Failed to connect to Redis`

Redis is not running or the gateway cannot reach it.

For local Docker Compose:

```bash
docker-compose ps
docker-compose logs redis
```

For local Windows Redis:

```powershell
redis\redis-cli.exe ping
```

Expected output:

```text
PONG
```

### `Node not found. Register your node first.`

The node ID was not registered before authentication. Register it first:

```bash
curl -X POST http://localhost:8000/api/admin/nodes \
  -H "X-Admin-Key: test-admin-key" \
  -H "Content-Type: application/json" \
  -d '{"node_id":"raspberry-pi-01","node_type":"residential"}'
```

### Peer Dashboard shows blank screen

Check browser console errors and verify the gateway is running:

```bash
curl http://localhost:8000/health
```

Expected response:

```json
{"status":"healthy","version":"1.0.0"}
```
