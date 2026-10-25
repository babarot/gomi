# gomi demo for the README: delete files, then browse and restore them
#
# Run with: mise run demo demo-readme
# Requirements: Docker

Output demo-readme.mp4

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
Type "ls" Sleep 350ms Enter Sleep 1s
Type "gomi debug.log scratch.txt" Sleep 350ms Enter Sleep 1s
Type "ls" Sleep 350ms Enter Sleep 1.5s

# Open the trash: it starts with what was deleted here
Type "gomi -b" Sleep 350ms Enter Sleep 2s
Down@400ms 4
Sleep 1s

# The preview of a file in full
Space Sleep 2s
Down@200ms 3
Sleep 500ms
Space Sleep 1s

# Every file, by the places they were deleted from, then by command
Type "." Sleep 2s
Type "v" Sleep 2s
Down@400ms 3
Type "l" Sleep 1.5s
Type "v" Sleep 2.5s

# The picker: type part of a name
Type "v" Sleep 1s
Type@150ms "dbg" Sleep 1.5s
Tab Sleep 500ms
Escape Sleep 300ms
Type "/" Sleep 300ms
Backspace@150ms 3
Type@150ms "scr" Sleep 1s
Tab Sleep 500ms
Enter Sleep 1s
Type "y" Sleep 500ms
Type "y" Sleep 1.5s

Type "ls" Sleep 350ms Enter Sleep 2.5s
