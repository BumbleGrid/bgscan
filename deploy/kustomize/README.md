# bgscan kustomize deploy

Customer-facing install instructions live in the repository root [README](../../README.md#deploy-to-kubernetes-kustomize).

Apply the default bundle:

```bash
kubectl apply -k deploy/kustomize/overlays/default
```

Layout:

- `base/` — Namespace, RBAC, Secret template, CronJob, manual Job
- `overlays/default/` — namespace override, image tag pin (`__BGSCAN_VERSION__` on `main`; substituted at release — see `scripts/release-version-files.txt`)
