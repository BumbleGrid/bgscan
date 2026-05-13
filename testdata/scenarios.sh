#!/usr/bin/env bash
# Apply/delete predefined Kubernetes cluster scenarios on kind.
#
# Scenario layout:
#   testdata/clusters/<scenario-name>/
#     scenario.env             # required; defines cluster settings
#     manifests/*.yaml         # default; resources to apply (see SCENARIO_MANIFESTS_DIR)
#
# Optional scenario.env variables:
#   SCENARIO_MANIFESTS_DIR   Subdirectory under the scenario folder (default: manifests).
#   SCENARIO_KUBECTL_RECURSIVE  If true, kubectl apply/delete uses --recursive (for nested YAML dirs).
#
# Example:
#   ./scenarios.sh list
#   ./scenarios.sh apply simple --recreate
#   ./scenarios.sh delete simple --delete-cluster

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCENARIOS_DIR="${SCRIPT_DIR}/clusters"
readonly PROVISION_SCRIPT="${SCRIPT_DIR}/provision-kind.sh"

ACTION=""
SCENARIO_NAME=""
RECREATE=false
DELETE_CLUSTER=false

usage() {
  cat <<EOF
Usage: $(basename "$0") <command> [scenario] [options]

Commands:
  list                              List available scenarios.
  apply <scenario> [--recreate]     Provision cluster + apply scenario manifests.
  delete <scenario> [--delete-cluster]
                                    Delete scenario resources; optionally delete cluster too.

Options:
  --recreate        Recreate cluster before applying scenario (apply only).
  --delete-cluster  Delete the scenario cluster after deleting resources (delete only).
  -h, --help        Show this help.

Examples:
  $(basename "$0") list
  $(basename "$0") apply simple
  $(basename "$0") apply complex --recreate
  $(basename "$0") delete simple
  $(basename "$0") delete complex --delete-cluster
EOF
}

log() { printf '[scenarios] %s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

list_scenarios() {
  local found=false
  local d
  for d in "${SCENARIOS_DIR}"/*; do
    [[ -d "$d" ]] || continue
    [[ -f "${d}/scenario.env" ]] || continue
    found=true
    basename "$d"
  done
  [[ "$found" == true ]] || log "no scenarios found in ${SCENARIOS_DIR}"
}

parse_args() {
  [[ $# -gt 0 ]] || { usage; exit 1; }
  case "${1:-}" in
    -h|--help|help)
      usage
      exit 0
      ;;
    list)
      ACTION="list"
      shift
      ;;
    apply|delete)
      ACTION="$1"
      shift
      SCENARIO_NAME="${1:-}"
      [[ -n "$SCENARIO_NAME" ]] || die "scenario name is required for '${ACTION}'"
      shift || true
      ;;
    *)
      die "unknown command: $1"
      ;;
  esac

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --recreate)
        RECREATE=true
        shift
        ;;
      --delete-cluster)
        DELETE_CLUSTER=true
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
}

load_scenario() {
  local name="$1"
  local dir="${SCENARIOS_DIR}/${name}"
  local env_file="${dir}/scenario.env"
  local manifests_subdir
  local manifests_dir

  [[ -d "$dir" ]] || die "scenario not found: $name"
  [[ -f "$env_file" ]] || die "missing file: $env_file"

  # shellcheck disable=SC1090
  source "$env_file"

  manifests_subdir="${SCENARIO_MANIFESTS_DIR:-manifests}"
  manifests_dir="${dir}/${manifests_subdir}"

  [[ -d "$manifests_dir" ]] || die "missing directory: $manifests_dir"

  : "${SCENARIO_CLUSTER_NAME:?SCENARIO_CLUSTER_NAME is required in scenario.env}"
  : "${SCENARIO_NODES:?SCENARIO_NODES is required in scenario.env}"
  : "${SCENARIO_NAMESPACES:?SCENARIO_NAMESPACES is required in scenario.env}"

  [[ "$SCENARIO_NODES" =~ ^[0-9]+$ ]] || die "SCENARIO_NODES must be numeric in $env_file"
  [[ "$SCENARIO_NODES" -ge 1 ]] || die "SCENARIO_NODES must be >= 1 in $env_file"
}

apply_scenario() {
  local name="$1"
  local dir="${SCENARIOS_DIR}/${name}"
  local manifests_subdir="${SCENARIO_MANIFESTS_DIR:-manifests}"
  local manifests_dir="${dir}/${manifests_subdir}"
  local context="kind-${SCENARIO_CLUSTER_NAME}"
  local recreate_flag=""
  local recursive_flag=""
  if [[ "$RECREATE" == true ]]; then
    recreate_flag="--recreate"
  fi
  if [[ "${SCENARIO_KUBECTL_RECURSIVE:-}" == "true" ]]; then
    recursive_flag="--recursive"
  fi

  log "provisioning scenario cluster '${SCENARIO_CLUSTER_NAME}'..."
  "${PROVISION_SCRIPT}" create \
    --clusters "${SCENARIO_CLUSTER_NAME}" \
    --nodes "${SCENARIO_NODES}" \
    --namespaces "${SCENARIO_NAMESPACES}" \
    ${recreate_flag}

  log "applying manifests from ${manifests_dir} to context ${context}..."
  kubectl --context "${context}" apply -f "${manifests_dir}" ${recursive_flag}

  log "scenario '${name}' applied successfully."
}

delete_scenario() {
  local name="$1"
  local dir="${SCENARIOS_DIR}/${name}"
  local manifests_subdir="${SCENARIO_MANIFESTS_DIR:-manifests}"
  local manifests_dir="${dir}/${manifests_subdir}"
  local context="kind-${SCENARIO_CLUSTER_NAME}"
  local recursive_flag=""
  if [[ "${SCENARIO_KUBECTL_RECURSIVE:-}" == "true" ]]; then
    recursive_flag="--recursive"
  fi

  if kubectl config get-contexts "${context}" >/dev/null 2>&1; then
    log "deleting manifests from ${manifests_dir} in context ${context}..."
    kubectl --context "${context}" delete -f "${manifests_dir}" ${recursive_flag} --ignore-not-found
  else
    log "context ${context} not found; skipping manifest deletion"
  fi

  if [[ "${DELETE_CLUSTER}" == true ]]; then
    log "deleting scenario cluster '${SCENARIO_CLUSTER_NAME}'..."
    "${PROVISION_SCRIPT}" delete --clusters "${SCENARIO_CLUSTER_NAME}"
  fi

  log "scenario '${name}' deleted."
}

main() {
  parse_args "$@"
  require_cmd kubectl

  if [[ "${ACTION}" == "list" ]]; then
    list_scenarios
    return 0
  fi

  load_scenario "${SCENARIO_NAME}"

  case "${ACTION}" in
    apply) apply_scenario "${SCENARIO_NAME}" ;;
    delete) delete_scenario "${SCENARIO_NAME}" ;;
    *) die "unsupported action: ${ACTION}" ;;
  esac
}

main "$@"
