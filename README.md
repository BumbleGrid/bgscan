# bgscan

Scan a Kubernetes cluster and emit a BGSpec Floor 0 JSON document. Optionally push the extraction to a BumbleGrid backend and save a local copy of the JSON.

## Docker

The image packages the `bgscan` CLI. Use it to scan a cluster, push the result to your BumbleGrid API (same as `output: push` on the CLI), and write the JSON to a mounted volume.

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
  bgscan:latest \
  --kubeconfig /kube/config \
  --output /output/bgdoc.json
```

Push to BumbleGrid and save a local copy. For local clusters (kind, minikube), add `--network host` on Linux (see **Run** above).

```bash
mkdir -p output

docker run --rm --network host \
  -v "$HOME/.kube/config:/kube/config:ro" \
  -v "$(pwd)/output:/output" \
  bgscan:latest \
  --kubeconfig /kube/config \
  --output push \
  --endpoint https://api.bumblegrid.tech \
  --org your-org-slug \
  --document your-document-slug \
  --cluster your-cluster-slug \
  --api-key "bg_sk_..." \
  --local-output /output/bgdoc.json
```

Optional: put the same settings in `bgconfig.yaml` and pass `--config /path/to/bgconfig.yaml` (see `bgconfig.sample.yaml`).

### Flags useful in Docker

| Flag | Purpose |
|------|---------|
| `--config` | Path to `bgconfig.yaml` inside the container |
| `--kubeconfig` | Path to kubeconfig (default: in-cluster, then `~/.kube/config`) |
| `--context` | Single kubeconfig context to scan |
| `--namespaces` | Comma-separated namespace filter |
| `--local-output` | Write the JSON to this path (in addition to push when `output: push`) |
| `--whole-document` | Emit full BGSpec document instead of floor 0 only |
| `--push-dry-run` | Print the resolved push request without sending it |

### Local file only (no push)

Omit push settings and set `output` to a path inside a mounted volume. For local clusters (kind, minikube), include `--network host` on Linux — see the example under **Run** above.

```bash
docker run --rm --network host \
  -v "$HOME/.kube/config:/kube/config:ro" \
  -v "$(pwd)/output:/output" \
  bgscan:latest \
  --kubeconfig /kube/config \
  --output /output/bgdoc.json
```

## CLI (without Docker)

```bash
go build -o bgscan .
./bgscan --help
```

Copy `bgconfig.sample.yaml` to `bgconfig.yaml` beside the binary, or pass `--config /path/to/bgconfig.yaml`.
