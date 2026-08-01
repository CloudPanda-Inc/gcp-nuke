#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

cd "$repo_root"

printf '%s\n' '==> go build'
go build -trimpath -o "$tmp_dir/gcp-nuke" .

printf '%s\n' '==> go vet'
go vet ./...

printf '%s\n' '==> go test ./resources/...'
go test ./resources/...

printf '%s\n' '==> resource type contract'
export NO_COLOR=1
export TERM=dumb
resource_types_output="$($tmp_dir/gcp-nuke resource-types)"

mapfile -t resource_types < <(
  printf '%s\n' "$resource_types_output" \
    | sed -nE 's/^[[:space:]]*([A-Z][A-Za-z0-9]+)[[:space:]]*$/\1/p' \
    | sort -u
)

required_resource_types=(
  CloudBuildTrigger
  DatabaseMigrationConnectionProfile
  DatabaseMigrationJob
)

for required in "${required_resource_types[@]}"; do
  if ! printf '%s\n' "${resource_types[@]}" | grep -Fxq "$required"; then
    printf 'missing fork-required resource type: %s\n' "$required" >&2
    printf '%s\n' 'resource-types output:' >&2
    printf '%s\n' "$resource_types_output" >&2
    exit 1
  fi
done

minimum_resource_type_count=91
if (( ${#resource_types[@]} < minimum_resource_type_count )); then
  printf 'resource type regression: got %d, expected at least %d\n' \
    "${#resource_types[@]}" "$minimum_resource_type_count" >&2
  printf '%s\n' 'resource-types output:' >&2
  printf '%s\n' "$resource_types_output" >&2
  exit 1
fi

printf 'verified %d resource types; required fork handlers are present\n' "${#resource_types[@]}"
