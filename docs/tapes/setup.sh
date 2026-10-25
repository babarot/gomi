#!/usr/bin/env bash
# Setup for every demo tape: install gomi, write its config, and fill the
# XDG trash with files deleted over the last weeks, so that the browser has
# days, places and batches to show. The tapes start in ~/projects/gomi.
#
# The trash is written by hand rather than with gomi, since gomi would
# stamp every file with the time of the recording.
set -euo pipefail

cp /data/.gomi /usr/local/bin/gomi
chmod +x /usr/local/bin/gomi

# The first run writes the default config. The default gomi_dir would make
# strategy auto read only the legacy trash, so ask for the XDG one.
gomi --version > /dev/null
config="$HOME/.config/gomi/config.yaml"
sed -i 's/strategy: auto/strategy: xdg/; /permanent_delete:/{n;s/enable: false/enable: true/}' "$config"

trash="$HOME/.local/share/Trash"
mkdir -p "$trash/files" "$trash/info"

# trashed <path> <ago> [run] writes <path> as deleted <ago> (a date -d
# offset such as "2 hours ago") by the command <run>. Files deleted by one
# command share a run. The content comes from stdin; a path ending in /
# is a directory, filled from stdin as "name<TAB>content" lines.
trashed() {
  local path="$1" ago="$2" run="${3:-}" name
  name="$(basename "$path")"
  if [[ "$path" == */ ]]; then
    mkdir -p "$trash/files/$name"
    while IFS=$'\t' read -r file content; do
      printf '%s\n' "$content" > "$trash/files/$name/$file"
    done
  else
    cat > "$trash/files/$name"
  fi
  {
    echo "[Trash Info]"
    echo "Path=${path%/}"
    echo "DeletionDate=$(date -d "$ago" +%Y-%m-%dT%H:%M:%S)"
    if [[ -n "$run" ]]; then echo "X-Gomi-RunID=$run"; fi
  } > "$trash/info/$name.trashinfo"
}

# The shells tmux starts take the prompt the tapes show
echo "PS1='\[\e[34m\]>\[\e[0m\] '" >> "$HOME/.bashrc"

# The project the tapes work in
work="$HOME/projects/gomi"
mkdir -p "$work/internal/ui" "$HOME/docs" "$HOME/Downloads" "$HOME/.config/nvim"
cd "$work"
printf 'module github.com/babarot/gomi\n\ngo 1.26\n' > go.mod
printf '# gomi\n\nA safer alternative to rm.\n' > README.md
cat > main.go << 'EOF'
package main

import "github.com/babarot/gomi/internal/cli"

func main() {
	cli.Run()
}
EOF
printf 'panic: runtime error\n' > debug.log
printf 'TODO: drop the old view\n' > scratch.txt

H=$'\t'

# Today
trashed "$work/internal/ui/old_view.go" "25 minutes ago" run-today << 'EOF'
package ui

// oldView draws the list the way it used to
func oldView(files []File) string {
	var b strings.Builder
	for _, f := range files {
		b.WriteString(f.Name + "\n")
	}
	return b.String()
}
EOF
trashed "$work/internal/ui/old_view_test.go" "25 minutes ago" run-today << 'EOF'
package ui

import "testing"

func TestOldView(t *testing.T) {
	if got := oldView(nil); got != "" {
		t.Errorf("oldView(nil) = %q", got)
	}
}
EOF
trashed "$HOME/Downloads/invoice-2026-10.txt" "3 hours ago" << 'EOF'
Invoice 2026-10

  Hosting        $12.00
  Domain          $1.20
  ---------------------
  Total          $13.20
EOF

# Yesterday
trashed "$work/coverage/" "1 day ago 3 hours ago" run-coverage << EOF
coverage.out${H}mode: count
coverage.html${H}<html><body>coverage</body></html>
EOF
trashed "$HOME/.config/nvim/init.lua.bak" "1 day ago 6 hours ago" << 'EOF'
vim.opt.number = true
vim.opt.expandtab = true
vim.opt.shiftwidth = 2
EOF

# Earlier this week: a project removed with its directory (+), and a note
# written again since (!)
trashed "$HOME/projects/old-app/README.md" "3 days ago 2 hours ago" run-old-app << 'EOF'
# old-app

The first try at a trash for the shell.
EOF
trashed "$HOME/projects/old-app/config.yaml" "3 days ago 2 hours ago" run-old-app << 'EOF'
trash:
  dir: ~/.old-app
  keep_days: 30
EOF
trashed "$HOME/docs/notes.md" "4 days ago 5 hours ago" << 'EOF'
# Notes

- Show the files deleted here first
- Group the files deleted by one command
EOF
printf '# Notes\n\nStarted again.\n' > "$HOME/docs/notes.md"

# Weeks ago
trashed "$work/Makefile" "12 days ago 1 hour ago" << 'EOF'
build:
	go build -o gomi .

test:
	go test ./...
EOF
trashed "$HOME/docs/draft.txt" "20 days ago 4 hours ago" << 'EOF'
A draft that did not make it.
EOF

# A record whose file is gone, for --prune=orphans
printf '[Trash Info]\nPath=%s\nDeletionDate=%s\n' "$HOME/tmp/lost.txt" \
  "$(date -d "40 days ago" +%Y-%m-%dT%H:%M:%S)" > "$trash/info/lost.txt.trashinfo"
