#!/usr/bin/env bash
# Record the demo GIFs with VHS in Docker and copy them to docs/demo, where
# the README and gomi.dev use them: docs/tapes/demo-<name>.tape becomes
# docs/demo/<name>.gif.
#
# Usage: demo.sh [-e KEY=VAL ...] [NAME ...]
#   NAME  a tape to record, without .tape (demo-readme, demo-main, ...); all when none
#   -e    an environment variable for the recording containers
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
tapes_dir="$root/docs/tapes"

names=()
args=("$@")
while [[ $# -gt 0 ]]; do
  case "$1" in
    -e) shift 2 ;;
    *) names+=("$1"); shift ;;
  esac
done

if ! docker info > /dev/null 2>&1; then
  echo "Docker is not running; start Docker and try again." >&2
  exit 1
fi

echo "Building gomi for Linux..."
# the vhs image runs on the Docker host's architecture
case "$(docker info --format '{{.Architecture}}')" in
  aarch64 | arm64) arch=arm64 ;;
  *) arch=amd64 ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -ldflags "$("$root/scripts/ldflags.sh")" -o "$tapes_dir/.gomi" "$root"
trap 'rm -f "$tapes_dir/.gomi"' EXIT

"$tapes_dir/vhs.sh" ${args[@]+"${args[@]}"}

if [[ ${#names[@]} -eq 0 ]]; then
  for tape in "$tapes_dir"/demo-*.tape; do
    names+=("$(basename "$tape" .tape)")
  done
fi

echo "Copying the GIFs to docs/demo/..."
for name in "${names[@]}"; do
  cp "$tapes_dir/$name.gif" "$root/docs/demo/${name#demo-}.gif"
done
