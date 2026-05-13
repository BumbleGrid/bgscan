# Testing Cluster Scenarios

This directory contains automation to build local kind clusters and apply scenario manifests.

## Files

- `provision-kind.sh`: low-level kind cluster create/delete helper.
- `scenarios.sh`: scenario runner (`list`, `apply`, `delete`).
- `scenarios/<name>/scenario.env`: cluster settings per scenario.
- `scenarios/<name>/manifests/*.yaml`: resources to deploy for that scenario.

## Available scenarios

- `simple`: single-node cluster with frontend, API, database, and Redis.
- `complex`: two-node cluster with multiple services, CronJob, StatefulSets, and Ingress.

## Usage

Run from `testing/`:

```bash
./scenarios.sh list
./scenarios.sh apply simple
./scenarios.sh apply complex --recreate
./scenarios.sh delete simple
./scenarios.sh delete complex --delete-cluster
```

`apply` does:

1. Creates or reuses the scenario's kind cluster.
2. Ensures namespaces from `scenario.env` exist.
3. Applies all manifests in `scenarios/<name>/manifests/`.

## Troubleshooting kind startup

If cluster creation fails during kubeadm init, run `provision-kind.sh` directly with:

```bash
./provision-kind.sh create --clusters staging --nodes 3 --recreate --retain --kind-image kindest/node:v1.30.6
```

- `--retain` keeps failed node containers for inspection.
- On failure, logs are exported to `testing/.kind-logs/<cluster>-<timestamp>/`.
- `--kind-image` lets you pin a node image compatible with your local Docker/kernel setup.

## Create a new scenario

1. Create `scenarios/<your-name>/`.
2. Add `scenario.env`:

```bash
SCENARIO_CLUSTER_NAME="my-scenario"
SCENARIO_NODES="1"
SCENARIO_NAMESPACES="apps,data"
```

3. Add Kubernetes YAML files under `manifests/`.
4. Run `./scenarios.sh apply <your-name>`.
