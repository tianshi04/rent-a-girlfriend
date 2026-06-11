# identity-service Helm Chart

This Helm chart deploys the Identity Service and its supporting Kubernetes resources.

## Supported resources
- Namespace (optional)
- ServiceAccount
- ConfigMap
- Secret
- Deployment
- Service
- HorizontalPodAutoscaler
- Istio Gateway
- Istio HTTPRoute
- Istio PeerAuthentication
- Istio AuthorizationPolicy

## Install

```bash
helm install identity-service ./services/identity-service/helm/identity-service \
  --namespace identity-service \
  --create-namespace
```

## Override values

Use custom values or environment-specific overrides:

```bash
helm install identity-service ./services/identity-service/helm/identity-service \
  --namespace identity-service \
  --create-namespace \
  -f values.yaml \
  -f values-prod.yaml
```

## Secrets

Sensitive fields should be provided through a secure mechanism rather than hardcoding in `values.yaml`.

Example:

```bash
helm upgrade identity-service ./services/identity-service/helm/identity-service \
  --namespace identity-service \
  --set-string secrets.DB_PASSWORD="secret" \
  --set-string secrets.GOOGLE_CLIENT_SECRET="secret"
```

## Notes

- The chart is designed to be installed into a single namespace.
- Istio resources are enabled by default.
- If you use a different ingress hostname, set `gateway.hostname` accordingly.
