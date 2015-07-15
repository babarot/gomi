[![](https://raw.githubusercontent.com/b4b4r07/screenshots/master/gomi/logo.png)][gomi]

[![](https://img.shields.io/badge/platform-OSX%20|%20Linux%20|%20Windows-ff69b4.svg?style=flat-square)][release]
[![](http://img.shields.io/github/release/b4b4r07/gomi.svg?style=flat-square)][release]
[![](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

`gomi` is a simple trash tool that works on CLI, written in Go

## Description

The concept of the trashcan does not exist in Command-line interface ([CLI](http://en.wikipedia.org/wiki/Command-line_interface)). If you have deleted an important file by mistake with the `rm` command, it would be difficult to restore. Then, it's this `gomi`. Unlike `rm` command, it is possible to easily restore deleted files because `gomi` have the trashcan for the CLI. It is also possible to work with trashcan for Graphical user interface ([GUI](http://en.wikipedia.org/wiki/Graphical_user_interface)).

***DEMO:***

[![DEMO](https://raw.githubusercontent.com/b4b4r07/screenshots/master/gomi/demo.gif)][gomi]

\*1 *gomi* means a trash in Japanese.

\*2 It was heavily inspired by [peco/peco](https://github.com/peco/peco) and [mattn/gof](https://github.com/mattn/gof).

## Features

- Easy to restore (thanks to [`peco`](https://github.com/peco/peco)-like interface)
- Quick preview feature
- Customizes [YAML format](http://www.yaml.org) configuration file
- Interacts nicely with [`Trash`](http://en.wikipedia.org/wiki/Trash_(computing))
- Supports [`Put Back`](http://www.mac-fusion.com/trash-tip-how-to-put-files-back-to-their-original-location/)
- A single binary
- Cross-platform CLI app 

### Quick Look

Before you restore a file that was discarded from the trashcan, `gomi` has a function that browse the contents of the file. It is almost the same as the [Quick Look](http://en.wikipedia.org/wiki/Quick_Look) of OS X.
If the discarded file is a directory, it is recursively scan its contents and make the files and subdirectories list.

To QuickLook, type the <kbd>C-q</kbd> in Restore mode. Available key map is here: [Keymap](#keymap)

[![QuickLook](https://raw.githubusercontent.com/b4b4r07/screenshots/master/gomi/quicklook.png)][gomi]

### Put Back

`gomi` supports [`Put Back`](http://www.mac-fusion.com/trash-tip-how-to-put-files-back-to-their-original-location/). Because it is possible to combine the GUI trashcan with `gomi`, it is possible to restore the discard file from the GUI menu. Currently it has supported OS X only.

[![Put Back](https://raw.githubusercontent.com/b4b4r07/screenshots/master/gomi/putback.gif)][gomi]

### Works on Windows

`gomi` is a Cross-platform application. Basically, you can also use Windows. In the future, it will be possible to combine the Recycle Bin with `gomi`. We welcome your pull request.

[![Windows](https://raw.githubusercontent.com/b4b4r07/screenshots/master/gomi/windows.gif)][gomi]

## Usage

Basic usage is...

1. **Remove!** Throw away the trash :package:

		$ gomi files

2. **Restore!** Scavenge the trash :mag:

		$ gomi -r

It is able to replace `rm` with `gomi`. However, on the characteristics of it, it dosen't have options such as `-f` and `-i` at all. The available option is `-r`.

To actually delete rather than the trash:

	$ gomi ~/.gomi/2015/05/01/gomi_file.13_55_01

Run twice.

To specify the location where you want to restore:

	$ gomi -r .

In the above example, it's restored to the current directory.

For more information, see `gomi --help`.

### Keymap

| Keys | Actions |
|------|-------|
| Enter      | Restore a file under the cursor or selected files |
| C-c, Esc   | Exit from Restore mode or Quick Look mode with success status |
| C-n, Down  | Move the selected line cursor to one line below |
| C-p, Up    | Move the selected line cursor to one line above |
| C-f, Right | Move caret forward 1 character |
| C-b, Left  | Move caret backward 1 character |
| C-a        | Move caret to the beginning of line |
| C-e        | Move caret to the end of line |
| BackSpace  | Delete one character backward |
| C-u        | Delete the characters under the cursor backward until the beginning of the line |
| C-w        | Delete one word backward |
| C-l        | Redraws the screen |
| C-q        | Toggle Quick Look |
| C-i, Tab   | Toggle showing help message about gomi |
| C-v        | Select multiple lines |
| C-\_        | Remove a file under the cursor or selected files |

## Installation

If you want to go the Go way (install in GOPATH/bin) and just want the command:

	$ go get -u github.com/b4b4r07/gomi

### Mac OS X / Homebrew

If you're on OS X and want to use [Homebrew](https://github.com/b4b4r07/homebrew-gomi):

	$ brew tap b4b4r07/gomi
	$ brew install gomi

### Binary only

Otherwise, download the binary from [GitHub Releases](https://github.com/b4b4r07/gomi/releases) and drop it in your `$PATH`.

- [OS X (386)](https://github.com/b4b4r07/gomi/releases/download/v0.1.2/gomi_darwin_386)
- [Linux (386)](https://github.com/b4b4r07/gomi/releases/download/v0.1.2/gomi_linux_386)
- [Windows (386)](https://github.com/b4b4r07/gomi/releases/download/v0.1.2/gomi_windows_386.exe)

## Setup

### Replace rm with gomi

Put something like this in your `~/.bashrc` or `~/.zshrc`:

```bash
alias rm="gomi"
```

This is recommended. By doing so, it is possible to prevent `rm` command from removing an important file.

### config.yaml

`gomi` read the YAML configuration such as the following from the `~/.gomi/config.yaml`. In ***ignore_files***, you can describe shell file name pattern that you do not want to add to history for restoration.

```yaml
root: ~/.gomi
ignore_files:
  - .DS_Store
  - "*~"
gomi_size: 1000000000 # 1GB
```

## Versus Other Trash Tools

- [andreafrancia/trash-cli](https://github.com/andreafrancia/trash-cli)

	> Command line interface to the freedesktop.org trashcan.

- [sindresorhus/trash](https://github.com/sindresorhus/trash)

	> Cross-platform command-line app for moving files and directories to the trash - A safer alternative to `rm`

:do_not_litter: Do not use litter, use [gomi](https://github.com/b4b4r07/gomi).

## License

[MIT](https://raw.githubusercontent.com/b4b4r07/dotfiles/master/doc/LICENSE-MIT.txt) © BABAROT (a.k.a. [b4b4r07](http://tellme.tokyo))


[release]: https://github.com/b4b4r07/gomi/releases
[license]: https://raw.githubusercontent.com/b4b4r07/dotfiles/master/doc/LICENSE-MIT.txt
[gomi]: http://b4b4r07.com/gomi