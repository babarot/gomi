Require tmux
Require gomi

Output demo-prune.gif

Set Shell "zsh"
Set FontSize 20
Set Width 1100
Set Height 900
Set Theme "tokyonight"
Set FontFamily "Hack Nerd Font"

# start
Sleep 0.5s

# prune orphaned metadata
Type "gomi --prune=orphans" Sleep 200ms
Enter
Sleep 1
Type "y" Sleep 200ms
Sleep 1s

Enter Sleep 300ms
Enter Sleep 300ms

# prune older files
Type "gomi --prune=1day,1week" Sleep 200ms
Enter
Sleep 1
Type "y" Sleep 200ms
Sleep 1
Type "Y" Sleep 0.5s
Type "E" Sleep 0.5s
Type "S" Sleep 0.5s
Backspace Sleep 0.5s
Type "S" Sleep 0.5s
Backspace Sleep 0.5s
Escape
Sleep 2
