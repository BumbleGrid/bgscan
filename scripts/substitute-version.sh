#!/usr/bin/env bash
set -euo pipefail

readonly PLACEHOLDER='__BGSCAN_VERSION__'
readonly FILE_LIST="${0%/*}/release-version-files.txt"

usage() {
  echo "Usage: $(basename "$0") <version>" >&2
  echo "       $(basename "$0") --check-placeholder" >&2
  exit 1
}

if [[ "${1:-}" == "--check-placeholder" ]]; then
  while IFS= read -r rel_path || [[ -n "${rel_path}" ]]; do
    [[ -z "${rel_path}" || "${rel_path}" == \#* ]] && continue
    if ! grep -q "${PLACEHOLDER}" "${rel_path}"; then
      echo "Expected ${PLACEHOLDER} in ${rel_path}" >&2
      exit 1
    fi
  done < "${FILE_LIST}"
  exit 0
fi

version="${1:-}"
if [[ -z "${version}" ]]; then
  usage
fi

if [[ "${version}" == "${PLACEHOLDER}" ]]; then
  echo "Refusing to substitute placeholder with itself" >&2
  exit 1
fi

while IFS= read -r rel_path || [[ -n "${rel_path}" ]]; do
  [[ -z "${rel_path}" || "${rel_path}" == \#* ]] && continue
  if [[ ! -f "${rel_path}" ]]; then
    echo "Missing file: ${rel_path}" >&2
    exit 1
  fi
  sed -i "s/${PLACEHOLDER}/${version}/g" "${rel_path}"
  if grep -q "${PLACEHOLDER}" "${rel_path}"; then
    echo "Placeholder remains in ${rel_path} after substitution" >&2
    exit 1
  fi
done < "${FILE_LIST}"
