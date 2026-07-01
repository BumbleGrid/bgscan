# bgscan kustomize deploy

Customer-facing install instructions live in the repository root [README](../../README.md#deploy-to-kubernetes-kustomize).

Recommended order (Secret before workloads):

```bash
kubectl apply -f deploy/kustomize/base/namespace.yaml

kubectl create secret generic bgscan-config \
  --from-file=bgscan.yaml=./bgscan.yaml \
  -n bumblegrid-system \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -k deploy/kustomize/overlays/workload

kubectl apply -f deploy/kustomize/base/job-manual.yaml -n bumblegrid-system
```

Layout:

- `base/` — Namespace, RBAC, CronJob manifests; `secret.yaml` and `job-manual.yaml` are templates applied separately (not in the kustomize bundle)
- `overlays/default/` — namespace + workloads + image tag pin (`__BGSCAN_VERSION__` on `main`; substituted at release — see `scripts/release-version-files.txt`)
- `overlays/workload/` — RBAC + CronJob only (assumes namespace and Secret already exist); used by BumbleGrid onboarding
