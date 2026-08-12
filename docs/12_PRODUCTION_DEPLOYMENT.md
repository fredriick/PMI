# Production Deployment Runbook

## Prerequisites

- Kubernetes cluster (1.24+) with `kubectl` configured
- Helm 3.12+ installed
- Terraform 1.5+ installed (if using Terraform modules)
- AWS CLI or GCP CLI configured (if using cloud provider)
- Docker registry access (ECR, GCR, or equivalent)
- Redis Sentinel or Redis Cluster for HA
- TLS certificates (or Let's Encrypt configured)

## Pre-Deployment Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Code is linted: `gofmt -l .` and `golangci-lint run`
- [ ] Security scan passes: `govulncheck ./...`
- [ ] Docker images built and pushed to registry
- [ ] `config.yaml` reviewed and secrets rotated
- [ ] mTLS certificates provisioned and valid
- [ ] Redis HA configured (Sentinel or Cluster)
- [ ] Helm values reviewed for production (`helm/values-production.yaml`)
- [ ] Terraform state backed up (if using Terraform)

## 1. Build and Push Docker Images

```bash
# Build images
docker build -t proxymesh/gateway:latest .
docker build -t proxymesh/matchmaker:latest ./matchmaker

# Push to registry
docker tag proxymesh/gateway:latest <registry>/proxymesh/gateway:v1.0.0
docker tag proxymesh/matchmaker:latest <registry>/proxymesh/matchmaker:v1.0.0
docker push <registry>/proxymesh/gateway:v1.0.0
docker push <registry>/proxymesh/matchmaker:v1.0.0
```

## 2. Deploy Infrastructure (Terraform)

```bash
cd terraform
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

## 3. Deploy Application (Helm)

```bash
# Add/update secrets in Kubernetes
kubectl create secret generic proxymesh-secrets \
  --from-literal=redis-password=<REDIS_PASSWORD> \
  --from-literal=jwt-secret=<JWT_SECRET> \
  --from-literal=api-key-secret=<API_KEY_SECRET> \
  --dry-run=client -o yaml | kubectl apply -f -

# Install/upgrade Helm release
helm upgrade --install proxymesh ./helm \
  -f helm/values-production.yaml \
  --namespace proxymesh --create-namespace \
  --set image.tag=v1.0.0 \
  --set image.registry=<registry>
```

## 4. Verify Deployment

```bash
# Check pod status
kubectl get pods -n proxymesh

# Check logs
kubectl logs -n proxymesh -l app=proxymesh-gateway --tail=100
kubectl logs -n proxymesh -l app=proxymesh-matchmaker --tail=100

# Port-forward for local verification
kubectl port-forward -n proxymesh svc/proxymesh-gateway 8000:8000

# Verify health endpoints
curl http://localhost:8000/health
curl http://localhost:8000/v1/health
curl http://localhost:8000/v1/metrics
```

## 5. Post-Deployment

- [ ] Verify all pods are `Running` and `Ready`
- [ ] Verify health endpoints return `{"status":"healthy"}`
- [ ] Verify metrics endpoint returns Prometheus metrics
- [ ] Verify WebSocket connections work (admin dashboard)
- [ ] Verify SSE telemetry stream works (peer dashboard)
- [ ] Check Prometheus alerts in Grafana
- [ ] Run smoke tests: `k6 run loadtest/k6-loadtest.js`
- [ ] Verify Redis failover works (if using Sentinel)

## Rollback Procedure

```bash
# Rollback to previous release
helm history proxymesh -n proxymesh
helm rollback proxymesh <previous-revision> -n proxymesh

# Or redeploy previous image
helm upgrade --install proxymesh ./helm \
  -f helm/values-production.yaml \
  --namespace proxymesh \
  --set image.tag=v0.9.0
```

## Troubleshooting

| Issue | Resolution |
|-------|-----------|
| Pods crash-looping | Check `kubectl logs` for config errors, verify secrets exist |
| Redis connection refused | Verify Redis Sentinel/Cluster endpoints and passwords |
| mTLS handshake failures | Verify certificates are not expired and match hostnames |
| High latency | Check Redis load, increase replica count, review rate limits |
| WebSocket disconnects | Check ingress timeout settings, increase proxy read timeout |
