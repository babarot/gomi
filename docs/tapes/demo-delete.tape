# gomi demo: delete files from the trash for good
#
# Run with: mise run demo demo-delete
# Requirements: Docker

Output demo-delete.mp4

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
Type "clear"
Enter
Sleep 500ms
Show
# --- Demo starts ---
Sleep 700ms
Type "gomi -b" Sleep 350ms Enter Sleep 2s
Type "." Sleep 1.5s
Down@400ms 3
Tab Sleep 500ms
Down@400ms 2
Tab Sleep 800ms
Type "D" Sleep 1.5s
Type@400ms "YES" Sleep 1s
Enter Sleep 2.5s
