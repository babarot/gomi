[![log](./img/gomi.png)](https://github.com/b4b4r07/gomi "./gomi -r")

[![GitHub Releases](https://img.shields.io/badge/platform-OSX%20|%20Linux%20|%20Windows-ff69b4.svg)](https://github.com/b4b4r07/gomi/releases "Works on OS X, Linux and Windows")
[![License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](https://raw.githubusercontent.com/b4b4r07/dotfiles/master/doc/LICENSE-MIT.txt "License")

`gomi` is a simple trash tool that works on CLI, written in Go

## Description

The concept of the trashcan does not exist in Command-line Interface (CLI). If you have deleted an important file by mistake with the `rm` command, it would be difficult to restore. Then, it's this `gomi`. Unlike `rm` command, it is possible to easily restore deleted files because `gomi` have the trashcan for the CLI. It is also possible to work with trashcan for Graphical user interface (GUI).

***DEMO:***

![demo](./img/gomi.gif)

*gomi* means the trash in Japanese.

## Features

- Easy to restore (thanks to [`peco`](https://github.com/peco/peco)-like interface)
- QuickLook removed files
- Customize [YAML format](http://www.yaml.org) configuration file
- Works with [`Trash`](http://en.wikipedia.org/wiki/Trash_(computing))
- Supports [`Put Back`](http://www.mac-fusion.com/trash-tip-how-to-put-files-back-to-their-original-location/)
- A single binary
- Cross-platform CLI app 

### QuickLook

Before you restore a file that was discarded from the trashcan, `gomi` has a function that browse the contents of the file. It is almost the same as the QuickLook of OS X.
If the discarded file is a directory, it is recursively scan its contents and make the files and subdirectories list.

To QuickLook, type the *C-q* in Restore mode. Available key map in QuickLook is here: [Keymap](#keymap)

![ql](./img/gomi_quicklook.png)

### Put Back

`gomi` supports [`Put Back`](http://www.mac-fusion.com/trash-tip-how-to-put-files-back-to-their-original-location/). Because it is possible to combine the GUI trashcan with `gomi`, it is possible to restore the discard file from the GUI menu. Currently it has supported OS X only.

![put back](./img/gomi_system.gif)

### Works on Windows

`gomi` is a Cross-platform application. Basically, you can also use Windows. In the future, it will be possible to combine the Recycle Bin with `gomi`. We welcome the pull request.

![windows](./img/gomi_windows.png)

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

| Key | Action |
|:---:|:---:|
| Enter | Restore under the cursor |
| C-c, Esc | Quit Restore mode or QuickLook |
| C-n, Down | Select Down |
| C-p, Up | Select Up |
| C-q | Toggle the QuickLook |

## Installation

If you want to go the Go way (install in GOPATH/bin) and just want the command:

	$ go get -u github.com/b4b4r07/gomi

### Mac OS X / Homebrew

If you're on OS X and want to use [Homebrew](https://brew.sh):

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

```
alias rm="gomi"
```

This is recommended. By doing so, it is possible to prevent `rm` command from removing an important file.

### config.yaml

`gomi` read the YAML configuration such as the following from the `~/.gomi/config.yaml`. In ***ignore_files***, you can describe shell file name pattern that you do not want to add to history for restoration.

```yaml
ignore_files:
  - .DS_Store
  - "*~"
```

## Versus Other Trash Tools

- [andreafrancia/trash-cli](https://github.com/andreafrancia/trash-cli)

	> Command line interface to the freedesktop.org trashcan.

- [sindresorhus/trash](https://github.com/sindresorhus/trash)

	> Cross-platform command-line app for moving files and directories to the trash - A safer alternative to `rm`

:do_not_litter: Do not use litter, use [gomi](https://github.com/b4b4r07/gomi).

## License

[MIT](https://raw.githubusercontent.com/b4b4r07/dotfiles/master/doc/LICENSE-MIT.txt) © BABAROT (a.k.a. [b4b4r07](http://tellme.tokyo))
