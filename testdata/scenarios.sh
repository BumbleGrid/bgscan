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
#   ./scenarios.sh test simple
#   ./scenarios.sh test all --recreate
#   ./scenarios.sh test istio --update-golden

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly BGSCAN_ROOT="${SCRIPT_DIR}/.."
readonly SCENARIOS_DIR="${SCRIPT_DIR}/clusters"
readonly PROVISION_SCRIPT="${SCRIPT_DIR}/provision-kind.sh"
readonly BGSCAN_BIN="${BGSCAN_ROOT}/bgscan"

ACTION=""
SCENARIO_NAME=""
RECREATE=false
DELETE_CLUSTER=false
UPDATE_GOLDEN=false
TEST_ALL=false

usage() {
  cat <<EOF
Usage: $(basename "$0") <command> [scenario] [options]

Commands:
  list                              List available scenarios.
  apply <scenario> [--recreate]     Provision cluster + apply scenario manifests.
  delete <scenario> [--delete-cluster]
                                    Delete scenario resources; optionally delete cluster too.
  test <scenario|all> [options]     Apply scenario, run bgscan, compare to golden output.

Options:
  --recreate        Recreate cluster before applying scenario (apply/test only).
  --delete-cluster  Delete the scenario cluster after deleting resources (delete/test only).
  --update-golden   Write normalized scan output to the scenario golden file (test only).
  --keep-cluster    Alias for default test behavior: leave cluster running after test.
  -h, --help        Show this help.

Examples:
  $(basename "$0") list
  $(basename "$0") apply simple
  $(basename "$0") apply complex --recreate
  $(basename "$0") delete simple
  $(basename "$0") delete complex --delete-cluster
  $(basename "$0") test simple
  $(basename "$0") test all --recreate
  $(basename "$0") test istio --update-golden
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
    apply|delete|test)
      ACTION="$1"
      shift
      SCENARIO_NAME="${1:-}"
      [[ -n "$SCENARIO_NAME" ]] || die "scenario name is required for '${ACTION}'"
      if [[ "$SCENARIO_NAME" == "all" && "$ACTION" == "test" ]]; then
        TEST_ALL=true
      fi
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
      --update-golden)
        UPDATE_GOLDEN=true
        shift
        ;;
      --keep-cluster)
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
  apply_scenario_manifests "${manifests_dir}" "${context}" "${recursive_flag}"

  log "scenario '${name}' applied successfully."
}

apply_scenario_manifests() {
  local manifests_dir="$1"
  local context="$2"
  local recursive_flag="$3"
  local crd_manifest
  local crd_name

  shopt -s nullglob
  local crd_manifests=("${manifests_dir}"/00-*.yaml)
  shopt -u nullglob

  if [[ ${#crd_manifests[@]} -gt 0 ]]; then
    log "applying CRD manifests before remaining resources..."
    kubectl --context "${context}" apply -f "${crd_manifests[@]}" ${recursive_flag}
    for crd_manifest in "${crd_manifests[@]}"; do
      while IFS= read -r crd_name; do
        [[ -n "$crd_name" ]] || continue
        log "waiting for CRD ${crd_name} to be established..."
        kubectl --context "${context}" wait --for=condition=Established "crd/${crd_name}" --timeout=120s
      done < <(grep -E '^  name:' "${crd_manifest}" | awk '{print $2}')
    done
  fi

  kubectl --context "${context}" apply -f "${manifests_dir}" ${recursive_flag}
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

find_gowork() {
  local dir="${BGSCAN_ROOT}"
  while [[ "${dir}" != "/" ]]; do
    if [[ -f "${dir}/go.work" ]]; then
      printf '%s' "${dir}/go.work"
      return 0
    fi
    dir="$(dirname "${dir}")"
  done
  return 1
}

resolve_gowork() {
  local gowork=""
  local generated="${SCRIPT_DIR}/.go.work.scenarios"
  local bgbase_mod="${BGSCAN_ROOT}/../bgbase/go.mod"

  if gowork="$(find_gowork)"; then
    printf '%s' "${gowork}"
    return 0
  fi

  if [[ -f "${bgbase_mod}" ]]; then
    cat > "${generated}" <<'EOF'
go 1.22.2

use (
	..
	../../bgbase
)
EOF
    printf '%s' "${generated}"
    return 0
  fi

  return 1
}

go_run_env() {
  local -a env_args=()
  local gowork=""
  if gowork="$(resolve_gowork)"; then
    env_args+=(GOWORK="${gowork}")
  else
    env_args+=(GOFLAGS=-mod=mod)
  fi
  printf '%s\0' "${env_args[@]}"
}

ensure_bgscan_binary() {
  local force_rebuild="${1:-false}"
  if [[ "${force_rebuild}" != true && -x "${BGSCAN_BIN}" ]]; then
    return 0
  fi
  log "building bgscan binary at ${BGSCAN_BIN}..."
  local -a env_args=()
  while IFS= read -r -d '' env_arg; do
    env_args+=("${env_arg}")
  done < <(go_run_env)
  (cd "${BGSCAN_ROOT}" && env "${env_args[@]}" go build -o "${BGSCAN_BIN}" .)
}

wait_for_scenario_ready() {
  local context="kind-${SCENARIO_CLUSTER_NAME}"
  local namespace
  IFS=',' read -ra namespace_list <<< "${SCENARIO_NAMESPACES}"
  for namespace in "${namespace_list[@]}"; do
    namespace="$(printf '%s' "$namespace" | xargs)"
    [[ -n "$namespace" ]] || continue
    if kubectl --context "${context}" get deployments -n "${namespace}" --no-headers 2>/dev/null | grep -q .; then
      log "waiting for deployments in namespace ${namespace}..."
      kubectl --context "${context}" wait deployment --all -n "${namespace}" \
        --for=condition=Available --timeout=300s
    fi
    if kubectl --context "${context}" get statefulsets -n "${namespace}" --no-headers 2>/dev/null | grep -q .; then
      log "waiting for statefulsets in namespace ${namespace} (non-fatal)..."
      kubectl --context "${context}" wait statefulset --all -n "${namespace}" \
        --for=condition=Ready --timeout=120s 2>/dev/null || \
        log "warning: statefulsets in ${namespace} not ready yet; continuing scan"
    fi
    if kubectl --context "${context}" get cronjobs -n "${namespace}" --no-headers 2>/dev/null | grep -q .; then
      log "waiting for cronjob pods in namespace ${namespace} (non-fatal)..."
      kubectl --context "${context}" wait --for=condition=Ready pod \
        -l "job-name" -n "${namespace}" --timeout=120s 2>/dev/null || true
    fi
    if kubectl --context "${context}" get jobs -n "${namespace}" --no-headers 2>/dev/null | grep -q .; then
      log "waiting for jobs in namespace ${namespace} (non-fatal)..."
      kubectl --context "${context}" wait --for=condition=Complete job --all -n "${namespace}" \
        --timeout=120s 2>/dev/null || \
        log "warning: jobs in ${namespace} not complete yet; continuing scan"
    fi
  done
}

run_bgscan() {
  local output_path="$1"
  local context="kind-${SCENARIO_CLUSTER_NAME}"
  log "running bgscan against context ${context}..."
  "${BGSCAN_BIN}" \
    --context "${context}" \
    --extractor-version "test-scenario" \
    --auto-arrangement none \
    --output "${output_path}"
}

normalize_and_check() {
  local input_path="$1"
  local golden_path="$2"
  local normalize_flags=(--input "${input_path}" --golden "${golden_path}")
  if [[ "${UPDATE_GOLDEN}" == true ]]; then
    normalize_flags+=(--update-golden)
  else
    normalize_flags+=(--check)
  fi
  local -a env_args=()
  while IFS= read -r -d '' env_arg; do
    env_args+=("${env_arg}")
  done < <(go_run_env)
  (cd "${BGSCAN_ROOT}" && env "${env_args[@]}" go run ./testdata/tools/normalize_floor0 "${normalize_flags[@]}")
}

test_scenario() {
  local name="$1"
  local output_path
  local golden_path
  local context="kind-${SCENARIO_CLUSTER_NAME}"

  output_path="$(mktemp "${TMPDIR:-/tmp}/bgscan-${name}.XXXXXX.json")"
  golden_path="${SCENARIOS_DIR}/${name}/expected/floor0.golden.json"
  mkdir -p "$(dirname "${golden_path}")"

  trap "rm -f '${output_path}'" RETURN

  apply_scenario "${name}"
  wait_for_scenario_ready
  run_bgscan "${output_path}"
  normalize_and_check "${output_path}" "${golden_path}"

  if [[ "${DELETE_CLUSTER}" == true ]]; then
    log "deleting scenario cluster '${SCENARIO_CLUSTER_NAME}'..."
    "${PROVISION_SCRIPT}" delete --clusters "${SCENARIO_CLUSTER_NAME}"
  else
    log "leaving cluster running (context ${context}); use delete or test --delete-cluster to tear down"
  fi

  log "scenario '${name}' test passed."
}

run_test_all() {
  local name
  local failed=false
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    log "=== testing scenario ${name} ==="
    if ! (
      load_scenario "${name}"
      test_scenario "${name}"
    ); then
      failed=true
      log "scenario '${name}' test failed"
    fi
  done < <(list_scenarios)
  [[ "$failed" == false ]] || die "one or more scenario tests failed"
}

main() {
  parse_args "$@"
  require_cmd kubectl

  if [[ "${ACTION}" == "list" ]]; then
    list_scenarios
    return 0
  fi

  if [[ "${ACTION}" == "test" ]]; then
    require_cmd kind
    require_cmd go
    ensure_bgscan_binary true
    if [[ "${TEST_ALL}" == true ]]; then
      run_test_all
      return 0
    fi
    load_scenario "${SCENARIO_NAME}"
    test_scenario "${SCENARIO_NAME}"
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
