# Secret and Certificate Rotation Policy

ProxyMesh supports API keys, mTLS, Redis credentials, and optional JWT signing. Production deployments should rotate these secrets on a regular schedule.

## Rotation Targets

| Secret | Location | Rotation Frequency | Notes |
|--------|----------|-------------------|-------|
| Redis password | Kubernetes Secret / environment | 90 days | Requires Redis restart or AUTH reload |
| mTLS CA certificate | Kubernetes Secret / mounted volume | 12 months | Keep old CA during transition |
| mTLS server certificate | Kubernetes Secret / mounted volume | 90 days | Restart gateway pods after update |
| mTLS client certificates | Kubernetes Secret / peer node config | 90 days | Rotate peer nodes in batches |
| API admin key | Environment variable / secret | 90 days | Update dashboards and CLI defaults |
| JWT signing key | Kubernetes Secret / environment | 90 days | Requires JWT validation restart |

## mTLS Rotation Process

1. Generate a new CA and server certificate.
2. Mount both old and new CA bundles during the transition window.
3. Deploy new certificates to gateway pods.
4. Roll gateway pods one at a time.
5. Rotate peer client certificates in batches.
6. Remove the old CA after all peers have rotated.

## Kubernetes Secret Rotation

Create or replace secrets:

```bash
kubectl create secret generic proxymesh-secrets \
  --from-literal=redis-password='replace-me' \
  --from-literal=jwt-secret='replace-me' \
  -n proxymesh
```

Restart affected pods:

```bash
kubectl rollout restart deployment/proxymesh-gateway -n proxymesh
```

## API Key Rotation

Admin API keys are currently configured through environment variables and dashboard defaults.

Recommended production flow:

1. Generate a new admin key.
2. Update Kubernetes secrets or environment variables.
3. Restart gateway pods.
4. Update dashboards and CLI tooling.
5. Revoke the old key after a short transition window.

## Redis Password Rotation

1. Update Redis with the new password.
2. Update the Kubernetes secret.
3. Restart gateway pods.
4. Verify connectivity with health checks.

## JWT Secret Rotation

1. Generate a new signing secret.
2. Update the secret.
3. Restart services that validate JWTs.
4. Allow existing tokens to expire naturally.
