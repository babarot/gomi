Require tmux
Require gomi

Output demo-debug.gif

Set Shell "zsh"
Set FontSize 20
Set Width 1100
Set Height 900
Set Theme "tokyonight"
Set FontFamily "Hack Nerd Font"

# https://github.com/charmbracelet/vhs/issues/433
Hide
  Type "tmux -f /dev/null -L test new-session -- bash" Enter
  Type "tmux split-window -d -v -- bash && \" Enter
  Type "tmux set status && \" Enter
  Type 'tmux setw pane-border-style "fg=0" && \' Enter
  Type 'tmux setw pane-active-border-style "fg=0"' Enter
  Type 'tmux resize-pane -U 5' Enter
  Sleep 0.5
  Ctrl+L
  Sleep 1
Show

Type "gomi --debug=live"
Enter
Sleep 1

# tmux bind to switch to next pane
Ctrl+B
Type o

Sleep 1
Type "gomi --restore"
Enter
Sleep 2
Type "j" Sleep 200ms
Type "j" Sleep 200ms
Type "j" Sleep 200ms
Type "j" Sleep 200ms
Type "j" Sleep 200ms
Type "j" Sleep 200ms
Sleep 2
