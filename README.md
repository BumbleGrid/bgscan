# bgscan

Scan a Kubernetes cluster and emit a BGSpec Floor 0 JSON document. Optionally push the extraction to a BumbleGrid backend and save a local copy of the JSON.

**Recommended:** deploy bgscan **inside your cluster** with the kustomize bundle below. BumbleGrid never receives your kubeconfig — the scanner runs with in-cluster ServiceAccount credentials and pushes results over HTTPS.

A Helm chart is planned for a later release; this repository ships kustomize only for now.

## Deploy to Kubernetes (kustomize)

Run bgscan as a CronJob in your cluster. It scans Floor 0 on a schedule and pushes extractions to BumbleGrid. During onboarding, run a one-off Job first so you do not wait for the next CronJob tick.

### Prerequisites

- Kubernetes cluster (1.24+) with `kubectl` configured for cluster-admin or equivalent install permissions
- Outbound HTTPS from the cluster to your BumbleGrid API endpoint (see `endpoint` in `bgscan.yaml`; production default is `https://api.bumblegrid.tech`)
- A BumbleGrid organization with a cluster connection created in onboarding or cluster settings
- Ability to pull `bumblegrid/bgscan` from Docker Hub (see [Troubleshooting](#troubleshooting) if pulls are rate-limited)

### 1. Connect a cluster in BumbleGrid

In the BumbleGrid UI, complete **Connect cluster** (onboarding step 5 or your org’s cluster settings). Save the cluster connection and note the one-time push API key — it is shown only at creation time. If you lose it, an org admin can mint a replacement token from organization settings.

BumbleGrid stores cluster metadata and the push API key reference; it does **not** store kubeconfig or in-cluster credentials.

### 2. Download `bgscan.yaml`

Download the generated `bgscan.yaml` from the UI. It includes your org, document, cluster slugs, `output: push`, API endpoint, and `api_key`. Keep this file off git and out of ticket systems — treat it like a secret.

Expected shape (placeholders shown):

```yaml
org: "your-org-slug"
document: "your-document-slug"
cluster:
  name: "prod-eu-west-1"
  slug: "prod-eu-west-1"
  environment: "production"
  namespaces: []
kubeconfig: ""
context: ""
namespaces: []
ignore_namespaces:
  - kube-system
  - bumblegrid-system
  # ... platform defaults from config/default_scan_filters.yaml
ignore_workloads: []
extractor_version: ""
output: "push"
endpoint: "https://api.bumblegrid.tech"
api_key: "bg_sk_..."
```

Empty `kubeconfig` and `context` mean in-cluster authentication via the pod ServiceAccount.

SaaS-generated `bgscan.yaml` includes `ignore_namespaces` and `ignore_workloads` from the pinned bgscan release defaults. Edit those keys in the in-cluster Secret to customize what bgscan skips (same as rotating the API key).

### 3. Apply namespace and create the Secret

Create the install namespace, then load your SaaS-generated config into a Secret (namespace must exist first):

```bash
kubectl apply -f deploy/kustomize/base/namespace.yaml

kubectl create secret generic bgscan-config \
  --from-file=bgscan.yaml=./bgscan.yaml \
  -n bumblegrid-system \
  --dry-run=client -o yaml | kubectl apply -f -
```

Do not commit `bgscan.yaml` or rendered Secret manifests to git. Encrypt Secrets at rest using your cluster’s etcd encryption or KMS integration — that is the cluster operator’s responsibility.

To rotate the API key later, update the Secret the same way; the next Job or CronJob run picks up the new key automatically.

### 4. Apply RBAC and CronJob

With the real Secret in place, install the workload overlay (ServiceAccount, ClusterRole, ClusterRoleBinding, CronJob — no Secret, no first-run Job):

```bash
kubectl apply -k deploy/kustomize/overlays/workload
```

This creates:

| Resource | Name | Purpose |
|----------|------|---------|
| ServiceAccount | `bgscan` | Pod identity |
| ClusterRole + ClusterRoleBinding | `bgscan` | Read-only scan permissions |
| CronJob | `bgscan` | Twice-daily scheduled scan |

The pinned container image tag lives in `deploy/kustomize/overlays/workload/kustomization.yaml` (`images.newTag`). On `main` that value is the placeholder `__BGSCAN_VERSION__`; the [Publish container image](.github/workflows/publish-image.yml) workflow substitutes it (and every other entry in `scripts/release-version-files.txt`) when cutting a release tag.

`deploy/kustomize/overlays/default` still applies namespace plus the same workloads in one step — useful for GitOps or manual installs that create the Secret separately. The onboarding UI uses the workload overlay after step 3 so re-applying manifests never overwrites your Secret.

### 5. Run the first scan

Start the one-off Job:

```bash
kubectl apply -f deploy/kustomize/base/job-manual.yaml -n bumblegrid-system
```

Alternatively, create a Job from the CronJob template:

```bash
kubectl create job "bgscan-manual-$(date +%s)" \
  --from=cronjob/bgscan \
  -n bumblegrid-system
```

Watch progress:

```bash
kubectl logs -n bumblegrid-system -l app.kubernetes.io/name=bgscan --tail=100 -f
kubectl get jobs -n bumblegrid-system
```

If the Job fails, see [Troubleshooting](#troubleshooting) before retrying.

### 6. Confirm extraction in BumbleGrid

Open **Extraction status** in onboarding (step 6) or your document’s extraction view. BumbleGrid detects the first push automatically — there is no separate “test connection” API. Once the push lands, continue review and commit in the UI.

### 7. Ongoing sync (CronJob)

The CronJob `bgscan` runs **`bgscan run --config /etc/bgscan/bgscan.yaml`** on schedule **`0 6,18 * * *`** (06:00 and 18:00 UTC). `concurrencyPolicy: Forbid` prevents overlapping scans.

Suspend scheduled runs (maintenance):

```bash
kubectl patch cronjob bgscan -n bumblegrid-system -p '{"spec":{"suspend":true}}'
```

Resume:

```bash
kubectl patch cronjob bgscan -n bumblegrid-system -p '{"spec":{"suspend":false}}'
```

### Customization

Patch the overlay or add your own kustomize layer:

| Setting | Location | Notes |
|---------|----------|-------|
| Image tag | `deploy/kustomize/overlays/workload/kustomization.yaml` (or `overlays/default`) → `images.newTag` | Published tags from [Publishing](#publishing-maintainers); `main` holds `__BGSCAN_VERSION__` until release |
| Namespace | overlay `namespace:` field + subject namespace in binding | Default `bumblegrid-system` |
| Cron schedule | `deploy/kustomize/base/cronjob.yaml` → `spec.schedule` | Cron syntax |
| CPU/memory | `cronjob.yaml` / `job-manual.yaml` pod `resources` | Raise limits on large clusters |
| Scan timeout | `activeDeadlineSeconds` on Job/CronJob | Default 1800s (30 minutes) |
| Namespace allowlist | `namespaces` in `bgscan.yaml` | Empty list = all accessible namespaces (before ignores) |
| Namespace denylist | `ignore_namespaces` in `bgscan.yaml` | Omitted = built-in platform defaults; `[]` = scan everything |
| Workload denylist | `ignore_workloads` in `bgscan.yaml` | Omitted = no workload filtering; glob patterns `namespace/kind/name` |

Default `ignore_namespaces` live in [`config/default_scan_filters.yaml`](config/default_scan_filters.yaml) (also vendored into the BumbleGrid monorepo when `BGSCAN_DEPLOY_REF` bumps). To scan platform namespaces too, set `ignore_namespaces: []`. Clusters using `ingress-nginx` or similar instead of a namespace literally named `ingress` should edit that entry.

Workload pattern examples (Floor 0 kinds: `deployments`, `statefulsets`, `daemonsets`, `cronjobs`, `jobs`, `services`, `ingresses`):

| Pattern | Matches |
|---------|---------|
| `payments/deployments/eppo*` | Deployments named `eppo…` in `payments` |
| `payments/*/eppo-*` | Any Floor 0 kind with name `eppo-…` in `payments` |
| `*/services/kube-dns` | `kube-dns` Service in any scanned namespace |

CLI overrides: `--ignore-namespaces`, `--ignore-workloads` (comma-separated).

### RBAC reference

The `bgscan` ClusterRole grants read-only verbs **`get`**, **`list`**, and **`watch`** on the resources bgscan needs for Floor 0 extraction. There is no access to Secrets, ConfigMaps, Pods, Nodes, ReplicaSets, or wildcard `*`.

| API group | Resources | Scope |
|-----------|-----------|-------|
| `""` (core) | `namespaces` | cluster |
| `""` (core) | `services` | namespaced |
| `apps` | `deployments`, `statefulsets`, `daemonsets` | namespaced |
| `batch` | `cronjobs`, `jobs` | namespaced |
| `networking.k8s.io` | `ingresses` | namespaced |
| `networking.istio.io` | `virtualservices`, `destinationrules`, `serviceentries` | namespaced (when Istio CRDs exist) |
| (non-resource) | `/api`, `/apis`, `/api/*`, `/apis/*` | discovery for Istio presence check |

This is least-privilege relative to the v1 Floor 0 graph: workload and ingress topology, optional Istio routing objects, and Services. ConfigMaps, Secrets, PVCs, NetworkPolicies, HPAs, and ReplicaSets are **not** extracted in v1 — the emitted graph is workload-focused.

The ServiceAccount in `bumblegrid-system` cannot read other Secrets cluster-wide.

### Troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| `Forbidden` listing resources | RBAC not applied or wrong ServiceAccount | Verify `ClusterRoleBinding` subject is `bumblegrid-system/bgscan`; re-apply kustomize |
| `Forbidden` on `/apis` discovery | Missing discovery rules | Ensure `clusterrole.yaml` non-resource URLs are present |
| Push fails with connection timeout | Egress blocked to BumbleGrid API | Allow HTTPS to `endpoint` from worker nodes; configure proxy if required |
| HTTP 401/403 on push | Invalid or rotated API key | Update Secret from a fresh `bgscan.yaml` or mint a new org token |
| `ImagePullBackOff` | Docker Hub rate limit or private registry | Add `imagePullSecrets` or mirror `bumblegrid/bgscan`; see [Publishing](#publishing-maintainers) |
| Job exceeds deadline | Very large cluster | Increase `activeDeadlineSeconds` and pod memory in an overlay |
| CronJob never runs | Suspended or controller issue | `kubectl get cronjob bgscan -n bumblegrid-system`; check `suspend` and controller logs |

Inspect pod logs:

```bash
kubectl logs -n bumblegrid-system -l app.kubernetes.io/name=bgscan --tail=200
```

---

## Local CLI (development and CI)

For ad-hoc scans from your workstation or CI, build or install the binary and run:

```bash
go build -o bgscan .
./bgscan run --config ./bgscan.yaml
```

Copy `bgconfig.sample.yaml` to `bgconfig.yaml` beside the binary, or pass `--config /path/to/bgconfig.yaml`. The root command without `run` still works for backward compatibility, but **`bgscan run`** is the documented entry point (matches in-cluster manifests).

Multi-kubeconfig fan-out and localhost clusters (kind, minikube) are local-CLI concerns. In-cluster deploy always scans the local cluster with an empty context.

## Docker (development and CI)

The image packages the `bgscan` CLI. Use it to scan a cluster, push the result to your BumbleGrid API (same as `output: push` in config), and write JSON to a mounted volume.

### Build

From the `bgscan` repository root:

```bash
docker build -t bgscan:latest .
```

`bgbase` is resolved from `go.mod` (`github.com/BumbleGrid/bgbase` at a tagged version), not from a local sibling checkout. For private module access during the image build, pass a GitHub token as a BuildKit secret (same pattern as the monorepo API image):

```bash
DOCKER_BUILDKIT=1 docker build \
  --secret id=github_token,src="$HOME/.github_token" \
  -t bgscan:latest .
```

### Run

Mount:

1. **Kubeconfig** — cluster credentials (or rely on in-cluster config when running as a Pod).
2. **Output directory** — where the JSON file should be written (`--local-output` or `--output`).

**Local clusters (kind, minikube, etc.):** kubeconfig usually points API servers at `127.0.0.1`. Inside a default Docker container that address is the container itself, not your host. On Linux, add `--network host` so bgscan shares the host network (same as `kubectl` on the machine). Example:

```bash
docker run --rm --network host \
  -v "$HOME/.kube/config:/kube/config:ro" \
  -v "$(pwd)/output:/output" \
  bgscan:latest run \
  --kubeconfig /kube/config \
  --output /output/bgdoc.json
```

Push to BumbleGrid and save a local copy. For local clusters (kind, minikube), add `--network host` on Linux (see **Run** above).

```bash
mkdir -p output

docker run --rm --network host \
  -v "$HOME/.kube/config:/kube/config:ro" \
  -v "$(pwd)/output:/output" \
  -v "$(pwd)/bgscan.yaml:/config/bgscan.yaml:ro" \
  bgscan:latest run \
  --config /config/bgscan.yaml \
  --local-output /output/bgdoc.json
```

Optional: put the same settings in `bgconfig.yaml` and pass `--config /path/to/bgconfig.yaml` (see `bgconfig.sample.yaml`).

### Flags useful in Docker

| Flag | Purpose |
|------|---------|
| `--config` | Path to `bgscan.yaml` / `bgconfig.yaml` inside the container |
| `--kubeconfig` | Path to kubeconfig (default: in-cluster, then `~/.kube/config`) |
| `--context` | Single kubeconfig context to scan |
| `--namespaces` | Comma-separated namespace allowlist |
| `--ignore-namespaces` | Comma-separated namespace denylist (default: built-in platform list) |
| `--ignore-workloads` | Comma-separated workload deny patterns (`namespace/kind/name` globs) |
| `--local-output` | Write the JSON to this path (in addition to push when `output: push`) |
| `--whole-document` | Emit full BGSpec document instead of floor 0 only |
| `--push-dry-run` | Print the resolved push request without sending it |

### Local file only (no push)

Omit push settings and set `output` to a path inside a mounted volume. For local clusters (kind, minikube), include `--network host` on Linux — see the example under **Run** above.

```bash
docker run --rm --network host \
  -v "$HOME/.kube/config:/kube/config:ro" \
  -v "$(pwd)/output:/output" \
  bgscan:latest run \
  --kubeconfig /kube/config \
  --output /output/bgdoc.json
```

## Publishing (maintainers)

Published images live on Docker Hub at `bumblegrid/bgscan`. Builds are triggered manually from the bgscan GitHub repository.

### Prerequisites

- Docker Hub repository `bumblegrid/bgscan` exists under the `bumblegrid` org.
- GitHub repository secrets configured:
  - `DOCKERHUB_USERNAME` — Docker Hub login for the `bumblegrid` org
  - `DOCKERHUB_TOKEN` — access token with push rights to `bumblegrid/bgscan`
  - `GO_MODULE_GITHUB_TOKEN` (optional) — PAT with `repo` read access for private `github.com/BumbleGrid/bgbase`; falls back to the workflow `GITHUB_TOKEN` when unset

### Publish a version

1. Open **Actions → Publish container image → Run workflow** in the bgscan repo.
2. Set **image_tag** to an immutable semver (e.g. `v1.0.0-beta.5`). Optionally enable **also_tag_latest** to update the moving `latest` pointer.

The workflow replaces `__BGSCAN_VERSION__` in every file listed in `scripts/release-version-files.txt`, commits that tree, builds the image, pushes to Docker Hub, and creates the matching git tag. The tagged commit carries the real version; `main` keeps the placeholder.

3. After the workflow succeeds, verify (substitute your tag):

```bash
docker pull bumblegrid/bgscan:<tag>
docker run --rm bumblegrid/bgscan:<tag> run --help
```

The image entrypoint is `bgscan` with no default command args — in-cluster CronJob and Job manifests supply `run --config …`.

Security-conscious deployments can pin by digest instead of tag.

Docker Hub applies pull rate limits to anonymous users; document cluster `imagePullSecrets` or Docker Hub login if customers hit limits.
