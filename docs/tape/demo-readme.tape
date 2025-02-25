Require gomi

Output demo.gif

Set Shell "zsh"
Set FontSize 24
Set Width 1800
Set Height 1200
Set Theme "tokyonight"
Set FontFamily "Hack Nerd Font"

Hide
Type "setopt interactivecomments" Enter
Type "alias ls='exa --group-directories-first'" Enter
Type "clear" Enter
Show

## Example 1
Sleep 1s
Type "# Let's move files to trash" Sleep 0.2s Enter Sleep 0.5s
Type "ls" Sleep 0.2s Enter Sleep 0.5s
Type "gomi ./internal ./main.go" Sleep 0.2s Enter Sleep 0.5s
Type "ls" Sleep 0.2s Enter Sleep 0.5s
Type "# Ok, removed." Sleep 0.2s Enter Sleep 0.5s
Type "# Let's restore files!" Sleep 0.2s Enter Sleep 0.5s
Type "gomi --restore" Sleep 0.2s Enter Sleep 0.5s

# See the UI
Sleep 1.5s

# file 1
Type " " Sleep 2s
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Sleep 0.5s
Type "G" Sleep 100ms
Sleep 0.5s
Type "g" Sleep 100ms
Sleep 1s
Type " " Sleep 100ms
Sleep 1s

# next file
Type "j" Sleep 100ms
Sleep 1s

# file 2
Type " " Sleep 2s
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Type "j" Sleep 100ms
Sleep 0.5s
Type "G" Sleep 100ms
Sleep 0.5s
Type "g" Sleep 100ms
Sleep 1s
Type " " Sleep 100ms
Sleep 1s

# restore
Type "k" Sleep 100ms
Sleep 1s
Tab@1s
Tab@1s
Sleep 0.5s
Enter
Sleep 0.5s
Type "# Let's check file is back" Sleep 0.2s Enter Sleep 0.5s
Type "ls" Sleep 0.2s Enter Sleep 0.5s
Type "# Now restored!" Sleep 0.2s Enter Sleep 0.5s
Sleep 2s
