# Kubernetes Deployment

ProxyMesh includes Kubernetes manifests for single-region production deployments.

## Deploy

```bash
kubectl apply -k k8s/
```

This applies:

- `namespace.yaml` - Creates the `proxymesh` namespace
- `configmap.yaml` - Provides gateway configuration
- `redis.yaml` - Deploys Redis with persistent storage
- `gateway.yaml` - Deploys the ProxyMesh gateway with 2 replicas
- `service.yaml` - Exposes HTTP on port 80 and gRPC on port 9000
- `ingress.yaml` - Optional NGINX Ingress route for `proxymesh.local`

## Build and Push Image

The manifests use this placeholder image:

```text
ghcr.io/fredriick/proxymesh:latest
```

Build and push your own image:

```bash
docker build -t ghcr.io/fredriick/proxymesh:latest -f Dockerfile.gateway .
docker push ghcr.io/fredriick/proxymesh:latest
```

Update the image in Kubernetes:

```bash
kubectl set image deployment/proxymesh-gateway -n proxymesh gateway=ghcr.io/fredriick/proxymesh:latest
kubectl rollout restart deployment/proxymesh-gateway -n proxymesh
```

## Verify

```bash
kubectl get pods -n proxymesh
kubectl get svc -n proxymesh
kubectl logs -n proxymesh deployment/proxymesh-gateway
```

Health checks:

```bash
curl http://localhost/health
curl http://localhost/v1/health
curl http://localhost/v1/metrics
```

If using the LoadBalancer service, replace `localhost` with the external IP:

```bash
kubectl get svc proxymesh-gateway -n proxymesh
```

## Scale

```bash
kubectl scale deployment/proxymesh-gateway -n proxymesh --replicas=3
```

## Rollback

```bash
kubectl rollout undo deployment/proxymesh-gateway -n proxymesh
```

## Notes

- The current binary contains both Gateway and Matchmaker logic. The Kubernetes manifests deploy the gateway binary with Redis as the shared state store.
- The gateway reads `config.yaml` from `/app/config.yaml`, which is mounted from the `proxymesh-config` ConfigMap.
- Redis is deployed with a `ReadWriteOnce` PVC for local data persistence.
