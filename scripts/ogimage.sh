#!/usr/bin/env bash
# Draw the share images of gomi.dev (og:image) from scripts/ogimage/og.html
# with headless Chrome: docs/ogimage/PATTERN.png, 1200x630 each. The page
# uses one of them as its og:image (see docs/index.html).
#
# Usage: ogimage.sh [PATTERN ...]
#   PATTERN  can, terminal or spit; all three when none
# CHROME names the Chrome or Chromium to run when it is not found.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
out_dir="$root/docs/ogimage"
patterns=("$@")
[[ ${#patterns[@]} -gt 0 ]] || patterns=(can terminal spit)

chrome="${CHROME:-}"
if [[ -z "$chrome" ]]; then
  for c in "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
    "/Applications/Chromium.app/Contents/MacOS/Chromium" \
    google-chrome chromium chromium-browser; do
    if command -v "$c" >/dev/null 2>&1 || [[ -x "$c" ]]; then
      chrome="$c"
      break
    fi
  done
fi
if [[ -z "$chrome" ]]; then
  echo "ogimage: no Chrome found; set CHROME to the browser to run" >&2
  exit 1
fi

# a profile of its own, so that a Chrome already open does not take the run
profile="$(mktemp -d)"
trap 'rm -rf "$profile"' EXIT
mkdir -p "$out_dir"

for pattern in "${patterns[@]}"; do
  case "$pattern" in
    can | terminal | spit) ;;
    *) echo "ogimage: no pattern $pattern; can, terminal or spit" >&2; exit 1 ;;
  esac
  # the time budget lets the web fonts and the terminal's preview come in
  "$chrome" --headless=new --disable-gpu --hide-scrollbars --no-first-run \
    --user-data-dir="$profile" --force-device-scale-factor=1 \
    --window-size=1200,630 --virtual-time-budget=6000 \
    --screenshot="$out_dir/$pattern.png" \
    "file://$root/scripts/ogimage/og.html?v=$pattern" 2>/dev/null
  echo "ogimage: wrote docs/ogimage/$pattern.png"
done
