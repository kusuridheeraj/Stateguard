# Kubernetes Beta Setup

## Target

- Kubernetes clusters in beta support mode

## Expected First-Release Flow

1. Install Stateguard cluster components and local control-plane integration where applicable.
2. Configure namespaces, workloads, and protected release boundaries.
3. Validate beta-safe enforcement around destructive delete or uninstall operations.
4. Review warnings and support limits before relying on production-like flows.

## Current Phase 5 Scaffolds

- Kubernetes manifest inspection via `stateguard kube inspect -f examples/kubernetes-beta/manifests.yaml`
- example manifests in `examples/kubernetes-beta/manifests.yaml`
- beta support remains inspection-focused, not full admission enforcement yet

## Notes

- Kubernetes support is included in the first public release as beta, not as the strongest support tier.
- Strong guarantees will depend on runtime maturity, adapter maturity, and configured persistence semantics.
