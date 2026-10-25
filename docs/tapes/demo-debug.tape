# gomi demo: follow the log while using gomi
#
# Run with: mise run demo demo-debug
# Requirements: Docker

Output demo-debug.mp4

Set Theme { "name": "gomi-dark", "black": "#16161e", "red": "#f7768e", "green": "#9ece6a", "yellow": "#e0af68", "blue": "#7aa2f7", "magenta": "#bb9af7", "cyan": "#7dcfff", "white": "#a9b1d6", "brightBlack": "#565f89", "brightRed": "#f7768e", "brightGreen": "#9ece6a", "brightYellow": "#e0af68", "brightBlue": "#7aa2f7", "brightMagenta": "#bb9af7", "brightCyan": "#7dcfff", "brightWhite": "#c0caf5", "background": "#1a1b26", "foreground": "#a9b1d6", "selection": "#283457", "cursor": "#c0caf5" }

Set Shell "bash"
Set FontSize 20
Set Width 1400
Set Height 900
Set Padding 20
Set WindowBar ""
Set TypingSpeed 50ms

# --- Hidden setup ---
Hide
Type "source /data/setup.sh"
Enter
Sleep 2s
Type `tmux -u -f /dev/null -L demo new-session \; split-window -v -l 40% \; select-pane -t 0 \; set status off`
Enter
Sleep 1.5s
Show
# --- Demo starts ---
Sleep 500ms
# the log in the lower pane, gomi in the upper one
Ctrl+B
Down
Sleep 300ms
Type "gomi --debug=live" Sleep 350ms Enter Sleep 1.5s
Ctrl+B
Up
Sleep 300ms
Type "gomi -b" Sleep 350ms Enter Sleep 2s
Down@500ms 3
Type "v" Sleep 1.5s
Type "v" Sleep 1.5s
Type "q" Sleep 1s
Type "gomi scratch.txt" Sleep 350ms Enter Sleep 2.5s
