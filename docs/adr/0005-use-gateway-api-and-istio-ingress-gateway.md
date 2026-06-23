# ADR 0005: Use Kubernetes Gateway API and Istio Ingress Gateway

**Status:** Accepted
**Date:** 2026-06-22

## Context

We need a unified, shared Ingress Gateway (`global-gateway`) for the microservices in the `Rent-a-Girlfriend` system. The gateway must:
1. Support both local development (Kind cluster) and production environments.
2. Route external HTTP/HTTPS traffic to internal microservices.
3. Align with Istio Ambient Mode (which is enabled in the cluster).
4. Be managed declaratively via Kustomize (avoiding custom bash script installations).

For local development on Kind, the host machine's ports (80/443) are mapped to the cluster nodes. Thus, the gateway pod running on the worker node must bind to `hostPort` and run on the node labeled `ingress-ready=true`.

We considered the following approaches:
* **Legacy Kubernetes Ingress:** Simple but lacks advanced routing, weight-based splitting, and native integration with Istio Ambient Mode.
* **Istio Legacy Gateway CRD:** Functional but legacy, and not the industry-standard path forward.
* **Kubernetes Gateway API (Auto-provisioned with customization):** The modern standard. Istio automatically provisions the Envoy-based Deployment and Service when a `Gateway` resource is defined. Customization (like adding `hostPort` for Kind) can be achieved natively using the `spec.infrastructure.parametersRef` ConfigMap introduced in Gateway API v1.

## Decision

We decided to use the **Kubernetes Gateway API** with **Istio Ingress Gateway** auto-provisioning. 

1. **Shared Gateway Resource:** Define a single `Gateway` resource (`global-gateway`) in the `istio-ingress` namespace.
2. **Local Customization via ConfigMap:** For `dev`, we override the auto-provisioned deployment using a `ConfigMap` referenced under `spec.infrastructure.parametersRef`. This configures `hostPort: 80` and `nodeSelector: ingress-ready: "true"` on the pod, and patches the service to type `ClusterIP` (instead of `LoadBalancer`) to avoid pending statuses on Kind.
3. **Production Configuration:** For `prod`, we keep the default `LoadBalancer` service type and apply a Kustomize patch to add the HTTPS listener (port 443) with TLS termination.
4. **GitOps & Declarative Directory Structure:**
   * `infra/k8s/base/global-gateway.yaml`: Contains the base Gateway resource.
   * `infra/k8s/envs/dev/global-gateway-config.yaml`: The customization ConfigMap for local port mapping.
   * `infra/k8s/envs/dev/kustomization.yaml`: Patches the base Gateway to link to the dev ConfigMap.
   * `infra/k8s/envs/prod/kustomization.yaml`: Patches the base Gateway to add the HTTPS listener.

## Consequences

### Positive Outcomes
* **No Redundant Proxies:** By using Istio's native auto-provisioning with the `parametersRef` customization, we avoid running duplicate proxy deployments (e.g. Helm-managed proxy + auto-provisioned proxy), saving CPU and memory resources on the local development machine.
* **Modern Standard:** Adhering to the Kubernetes Gateway API ensures compatibility with future Kubernetes and service mesh standards.
* **Cleaner Repository Structure:** Environmental settings are fully encapsulated in their respective `dev` and `prod` folders, complying with Kustomize's load restrictions.

### Trade-offs
* **Istio Version Requirement:** Using `spec.infrastructure` parameter patching requires Istio 1.22+. Since our cluster is running Istio 1.29.2, this is fully supported.
* **Gateway API CRD Dependency:** Requires the Gateway API Custom Resource Definitions (`gateway.networking.k8s.io`) to be installed in the cluster (which is standard when installing Istio 1.22+).
