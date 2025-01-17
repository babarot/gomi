<p align="center">
  <img src="./docs/screenshot.png" width="500" alt="gomi">
</p>

<p align="center">
    <a href="https://b4b4r07.mit-license.org">
        <img src="https://img.shields.io/github/license/babarot/gomi" alt="License"/>
    </a>
    <a href="https://github.com/babarot/gomi/releases">
        <img
            src="https://img.shields.io/github/v/release/babarot/gomi"
            alt="GitHub Releases"/>
    </a>
    <br />
    <a href="https://babarot.github.io/gomi/">
        <img
            src="https://img.shields.io/website?down_color=lightgrey&down_message=donw&up_color=green&up_message=up&url=https%3A%2F%2Fbabarot.me%2Fgomi"
            alt="Website"
            />
    </a>
    <a href="https://github.com/babarot/gomi/actions/workflows/release.yaml">
        <img
            src="https://github.com/babarot/gomi/actions/workflows/release.yaml/badge.svg"
            alt="GitHub Releases"
            />
    </a>
    <a href="https://github.com/babarot/gomi/blob/master/go.mod">
        <img
            src="https://img.shields.io/github/go-mod/go-version/babarot/gomi"
            alt="Go version"
            />
    </a>
</p>

# 🗑️ Replacement for UNIX rm command!

`gomi` (ごみ/go-mi means a trash in Japanese) is a simple trash tool that works on CLI, written in Go

The concept of the trashcan does not exist in Command-line interface ([CLI](http://en.wikipedia.org/wiki/Command-line_interface)). If you have deleted an important file by mistake with the `rm` command, it would be difficult to restore. Then, it's this `gomi`. Unlike `rm` command, it is possible to easily restore deleted files because `gomi` have the trashcan for the CLI.

## Features

- Like a `rm` command but not unlink (delete) in fact (just move to another place)
- Easy to restore, super intuitive
- Compatible with `rm` command, e.g. `-r`, `-f` options
- Nice UI, awesome CLI UX
- Easy to see what gomi does with setting `GOMI_LOG=[trace|debug|info|warn|error]`

## Usage

```console
$ alias rm=gomi
```
```console
$ rm -rf important-dir
```
```console
$ rm --restore
Search: █
Which to restore?
  ▸ important-dir
    main_test.go
    main.go
    test-dir
↓   validate_test.rego

Name:             important-dir
Path:             /Users/babarot/src/github.com/babarot/important-dir
DeletedAt:        5 days ago
Content:            (directory)
  -rw-r--r--  important-file-1
  -rw-r--r--  important-file-2
  drwxr-xr-x  important-subdir-1
  drwxr-xr-x  important-subdir-2
  ...
```

## Installation

Download the binary from [GitHub Releases][release] and drop it in your `$PATH`.

- [Darwin / Mac][release]
- [Linux][release]

For macOS / [Homebrew](https://brew.sh/) user:

```bash
brew install babarot/tap/gomi
```

Using [afx](https://github.com/babarot/afx), package manager for CLI:

```yaml
github:
- name: babarot/gomi
  description: Trash can in CLI
  owner: babarot
  repo: gomi
  release:
    name: gomi
    tag: v1.1.5
  command:
    link:
    - from: gomi
    alias:
      rm: gomi
```


AUR users:

https://aur.archlinux.org/packages/gomi/

## Versus

- [andreafrancia/trash-cli](https://github.com/andreafrancia/trash-cli)
- [sindresorhus/trash](https://github.com/sindresorhus/trash)

## License

[MIT][license]

[release]: https://github.com/babarot/gomi/releases/latest
[license]: https://b4b4r07.mit-license.org
