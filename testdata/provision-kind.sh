#!/usr/bin/env bash
# Provision or tear down local Kubernetes clusters using kind (KinD).
# Suitable as a base for further scripts that add namespaces, workloads, etc.
#
# Requires: kind, kubectl
#
# Examples:
#   ./provision-kind.sh create
#   ./provision-kind.sh create --clusters demo,staging --nodes 3 --namespaces apps,monitoring
#   ./provision-kind.sh delete --clusters demo,staging

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly KIND_CONFIG_DIR="${SCRIPT_DIR}/.kind-config"
readonly DEFAULT_CLUSTER="kind"

ACTION="create"
CLUSTERS="${DEFAULT_CLUSTER}"
NODES="1"
NAMESPACES=""
RECREATE=false
RETAIN_ON_FAILURE=false
KIND_IMAGE="${KIND_IMAGE:-}"
LOG_DIR="${SCRIPT_DIR}/.kind-logs"
SKIP_HOST_PREFLIGHT=false

usage() {
  cat <<EOF
Usage: $(basename "$0") <command> [options]

Commands:
  create    Create kind cluster(s), wait until ready, optionally create namespaces (default).
  delete    Delete kind cluster(s) named by --clusters.

Options:
  --clusters LIST   Comma-separated kind cluster names (default: ${DEFAULT_CLUSTER}).
  --nodes N         Total nodes per cluster. N=1 is a single control-plane node.
                    N>=2 adds worker nodes (1 control-plane + N-1 workers).
  --namespaces LIST Comma-separated namespace names to create in each cluster (create only).
  --recreate        If cluster exists, delete it first, then create (create only).
  --kind-image IMG  kindest/node image to use (optional).
  --retain          Keep failed cluster nodes for debugging.
  --log-dir PATH    Where to export kind logs on failures (default: ${LOG_DIR}).
  --skip-preflight  Skip host inotify preflight checks.

Environment:
  KIND_CLUSTER_NAME is not used by this script; names come from --clusters only.

Examples:
  $(basename "$0") create
  $(basename "$0") create --clusters east,west --nodes 3 --namespaces team-a,team-b
  $(basename "$0") create --clusters staging --nodes 3 --kind-image kindest/node:v1.30.6 --retain
  $(basename "$0") delete --clusters east,west
EOF
}

log() { printf '[provision-kind] %s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

read_sysctl_value() {
  local key="$1"
  local v=""
  v="$(sysctl -n "$key" 2>/dev/null || true)"
  printf '%s' "$v"
}

host_preflight() {
  # kubelet/cAdvisor may fail inside kind nodes when host inotify limits are low.
  local min_instances=512
  local min_watches=262144
  local instances watches
  instances="$(read_sysctl_value fs.inotify.max_user_instances)"
  watches="$(read_sysctl_value fs.inotify.max_user_watches)"

  [[ -n "$instances" && -n "$watches" ]] || {
    log "warning: unable to read inotify sysctl values; skipping preflight check"
    return 0
  }

  if [[ "$instances" -lt "$min_instances" || "$watches" -lt "$min_watches" ]]; then
    die "host inotify limits are too low for reliable kind kubelet startup:
  fs.inotify.max_user_instances=${instances} (recommended >= ${min_instances})
  fs.inotify.max_user_watches=${watches} (recommended >= ${min_watches})

Temporarily apply:
  sudo sysctl -w fs.inotify.max_user_instances=${min_instances}
  sudo sysctl -w fs.inotify.max_user_watches=${min_watches}

Persist across reboots:
  echo 'fs.inotify.max_user_instances=${min_instances}' | sudo tee /etc/sysctl.d/99-kind-inotify.conf
  echo 'fs.inotify.max_user_watches=${min_watches}' | sudo tee -a /etc/sysctl.d/99-kind-inotify.conf
  sudo sysctl --system

If you still want to proceed without this check, pass --skip-preflight."
  fi
}

parse_list() {
  # trim, drop empty entries
  local IFS=','
  read -ra _arr <<<"${1:-}"
  local out=()
  local x
  for x in "${_arr[@]}"; do
    x="${x#"${x%%[![:space:]]*}"}"
    x="${x%"${x##*[![:space:]]}"}"
    [[ -n "$x" ]] && out+=("$x")
  done
  printf '%s\n' "${out[@]}"
}

kind_context() {
  local name="$1"
  echo "kind-${name}"
}

write_kind_config() {
  local path="$1"
  local total_nodes="$2"
  mkdir -p "$(dirname "$path")"
  if [[ "$total_nodes" -lt 1 ]]; then
    die "--nodes must be >= 1"
  fi
  {
    echo "kind: Cluster"
    echo "apiVersion: kind.x-k8s.io/v1alpha4"
    echo "nodes:"
    echo "  - role: control-plane"
    local w
    for ((w = 1; w < total_nodes; w++)); do
      echo "  - role: worker"
    done
  } >"$path"
}

cluster_exists() {
  local name="$1"
  kind get clusters 2>/dev/null | grep -qx "$name"
}

create_one_cluster() {
  local name="$1"
  local total_nodes="$2"
  local cfg="${KIND_CONFIG_DIR}/${name}.yaml"
  write_kind_config "$cfg" "$total_nodes"

  if cluster_exists "$name"; then
    if [[ "$RECREATE" == true ]]; then
      log "recreating existing cluster: $name"
      kind delete cluster --name "$name"
    else
      log "cluster already exists, skipping: $name (use --recreate to replace)"
      return 0
    fi
  fi

  log "creating cluster '$name' (${total_nodes} node(s))..."
  local create_args=(create cluster --name "$name" --config "$cfg" --wait 5m)
  if [[ -n "$KIND_IMAGE" ]]; then
    create_args+=(--image "$KIND_IMAGE")
  fi
  if [[ "$RETAIN_ON_FAILURE" == true ]]; then
    create_args+=(--retain)
  fi

  set +e
  kind "${create_args[@]}"
  local create_rc=$?
  set -e
  if [[ "$create_rc" -ne 0 ]]; then
    mkdir -p "$LOG_DIR"
    local ts
    ts="$(date +%Y%m%d-%H%M%S)"
    local cluster_log_dir="${LOG_DIR}/${name}-${ts}"
    log "cluster creation failed for '$name'. exporting logs to ${cluster_log_dir}..."
    kind export logs "$cluster_log_dir" --name "$name" >/dev/null 2>&1 || true
    die "kind create failed for '$name'. Check logs under ${cluster_log_dir}. Try --retain and/or --kind-image kindest/node:<version>."
  fi

  local ctx
  ctx="$(kind_context "$name")"
  log "waiting for nodes Ready in context $ctx..."
  kubectl --context "$ctx" wait --for=condition=Ready nodes --all --timeout=300s
}

create_namespaces_for_cluster() {
  local name="$1"
  local ns_csv="$2"
  [[ -z "$ns_csv" ]] && return 0

  local ctx
  ctx="$(kind_context "$name")"
  local ns
  while IFS= read -r ns; do
    [[ -z "$ns" ]] && continue
    if kubectl --context "$ctx" get namespace "$ns" &>/dev/null; then
      log "namespace exists, skipping: $ns (cluster $name)"
      continue
    fi
    log "creating namespace '$ns' in cluster '$name'..."
    kubectl --context "$ctx" create namespace "$ns"
  done < <(parse_list "$ns_csv")
}

delete_one_cluster() {
  local name="$1"
  if cluster_exists "$name"; then
    log "deleting cluster: $name"
    kind delete cluster --name "$name"
  else
    log "cluster not found, skipping: $name"
  fi
}

parse_args() {
  if [[ $# -eq 0 ]]; then
    usage
    exit 0
  fi
  case "${1:-}" in
    -h|--help|help)
      usage
      exit 0
      ;;
    create|delete)
      ACTION="$1"
      shift
      ;;
    *)
      die "unknown command: $1 (expected create or delete)"
      ;;
  esac

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --clusters)
        CLUSTERS="${2:-}"
        shift 2
        ;;
      --nodes)
        NODES="${2:-}"
        shift 2
        ;;
      --namespaces)
        NAMESPACES="${2:-}"
        shift 2
        ;;
      --recreate)
        RECREATE=true
        shift
        ;;
      --retain)
        RETAIN_ON_FAILURE=true
        shift
        ;;
      --kind-image)
        KIND_IMAGE="${2:-}"
        shift 2
        ;;
      --log-dir)
        LOG_DIR="${2:-}"
        shift 2
        ;;
      --skip-preflight)
        SKIP_HOST_PREFLIGHT=true
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        die "unknown option: $1"
        ;;
    esac
  done

  [[ "$ACTION" != "delete" ]] || NAMESPACES=""
}

main() {
  parse_args "$@"

  require_cmd kind
  require_cmd kubectl
  require_cmd sysctl

  if [[ ! "$NODES" =~ ^[0-9]+$ ]]; then
    die "--nodes must be a positive integer"
  fi
  [[ "$NODES" -ge 1 ]] || die "--nodes must be >= 1"

  if [[ "$ACTION" == "create" && "$SKIP_HOST_PREFLIGHT" != true ]]; then
    host_preflight
  fi

  local cluster_names=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && cluster_names+=("$line")
  done < <(parse_list "$CLUSTERS")

  [[ "${#cluster_names[@]}" -gt 0 ]] || die "no cluster names resolved from --clusters"

  local c
  if [[ "$ACTION" == "delete" ]]; then
    for c in "${cluster_names[@]}"; do
      delete_one_cluster "$c"
    done
    log "delete complete."
    return 0
  fi

  for c in "${cluster_names[@]}"; do
    create_one_cluster "$c" "$NODES"
  done

  if [[ -n "$NAMESPACES" ]]; then
    for c in "${cluster_names[@]}"; do
      create_namespaces_for_cluster "$c" "$NAMESPACES"
    done
  fi

  log "create complete. kubectl contexts:"
  for c in "${cluster_names[@]}"; do
    printf '  %s → cluster %s\n' "$(kind_context "$c")" "$c"
  done
}

main "$@"
