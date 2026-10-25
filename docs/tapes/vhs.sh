#!/usr/bin/env bash
# Record all VHS tapes in parallel, then convert MP4→GIF sequentially.
#
# Each container gets resource limits to prevent starvation when running
# in parallel. GIF generation is done separately via ffmpeg because VHS's
# built-in GIF output is unreliable under parallel Docker on macOS.
#
# Usage: vhs.sh [-e KEY=VAL ...] [NAME ...]
#   NAME  a tape to record, without .tape (demo-readme, demo-main, ...); all when none
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VHS_IMAGE="gomi-vhs"
FFMPEG_IMAGE="jrottenberg/ffmpeg:7-alpine"

docker_flags=()
tapes=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -e)
      docker_flags+=("$1" "$2")
      shift 2
      ;;
    *)
      if [[ ! -f "$SCRIPT_DIR/$1.tape" ]]; then
        echo "No tape named $1 (looked for $SCRIPT_DIR/$1.tape)" >&2
        exit 2
      fi
      tapes+=("$SCRIPT_DIR/$1.tape")
      shift
      ;;
  esac
done

if [[ ${#tapes[@]} -eq 0 ]]; then
  tapes=("$SCRIPT_DIR"/demo-*.tape)
fi

# Build the image with tmux added (cached after the first build)
docker build -q -t "$VHS_IMAGE" "$SCRIPT_DIR"

echo "Recording ${#tapes[@]} tapes in parallel ..."

pids=()
names=()
for tape in "${tapes[@]}"; do
  name="$(basename "$tape" .tape)"
  names+=("$name")
  echo "  ▶ $name"
  docker run --rm \
    --memory=2g --cpus=2 \
    -v "$SCRIPT_DIR":/data \
    -w /data \
    ${docker_flags[@]+"${docker_flags[@]}"} \
    "$VHS_IMAGE" "$name.tape" &
  pids+=($!)
done

failures=0
for i in "${!pids[@]}"; do
  name="${names[$i]}"
  if wait "${pids[$i]}"; then
    echo "  ✓ $name"
  else
    echo "  ✗ $name"
    failures=$((failures + 1))
  fi
done

if [[ $failures -gt 0 ]]; then
  echo "ERROR: $failures tape(s) failed" >&2
  exit 1
fi

# Re-generate GIFs from MP4 sequentially
echo "Converting MP4 → GIF ..."
for name in "${names[@]}"; do
  mp4="$SCRIPT_DIR/$name.mp4"
  if [[ -f "$mp4" && "$(stat -f%z "$mp4" 2>/dev/null || stat -c%s "$mp4")" -gt 0 ]]; then
    echo "  ▶ $name.gif"
    docker run --rm -v "$SCRIPT_DIR":/data -w /data "$FFMPEG_IMAGE" \
      -y -i "$name.mp4" \
      -vf "fps=10,scale=1200:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=256[p];[s1][p]paletteuse=dither=sierra2_4a" \
      "$name.gif" > /dev/null 2>&1
    echo "  ✓ $name.gif"
  fi
done

echo "${#tapes[@]} tape(s) recorded."
