# gomi

`gomi` is a simple trash script that works on CLI, written in Go

## Description

`rm` command threaten the CLI beginner. This is because it would delete the file without going through the trash. I was re-invented the `rm` as delete command with the concept of trash. It's `gomi`. Even if you delete the file by mistake, it can easily be restored.

***DEMO:***

![demo](./img/gomi.gif)

Incidentally, *gomi* means the trash in Japanese.

## Features

- `rm` command with the trash
- Easy to restore (thanks to [`peco`](https://github.com/peco/peco)-like UI)
- **QuickLook** files in Restore mode
- A single binary

### QuickLook

`gomi` has QuickLook (almost the same QuickLook of OS X) that can look through the file a little.

To QuickLook under the cursor file, type the *C-q* in Restore mode.

## Requirement

- Go

## Usage

1. **Remove!** Throw away the trash files :package:

		$ gomi files

2. **Restore!** Scavenge the trash box :mag:

		$ gomi -r

-----

To actually delete:

	$ gomi ~/.gomi/2015/05/01/gomi_file.13_55_01

Run twice.

To specify the location where you want to restore:

	$ gomi -r .

In the above example, it's restored to the current directory.

### Keymap

| Key | Action |
|:---:|:---:|
| Enter | Restore under the cursor |
| Esc | Quit Restore mode or QuickLook |
| C-c | Same as *Esc* |
| C-n | Select Down |
| C-p | Select Up |
| C-q | Toggle the QuickLook |

## Installation

	$ go get -u github.com/b4b4r07/gomi

## Setup

Put something like this in your ~/.bashrc or ~/.zshrc:

```
alias rm="gomi"
```

## Author

BABAROT a.k.a. [@b4b4r07](https://twitter.com/b4b4r07)

## License

[MIT](https://raw.githubusercontent.com/b4b4r07/dotfiles/master/doc/LICENSE-MIT.txt)
