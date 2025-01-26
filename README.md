<!-- <p align="center"> -->
<!--   <img src="./docs/screenshot.png" width="500" alt="gomi"> -->
<!-- </p> -->
<!---->
<!-- <p align="center"> -->
<!--     <a href="https://b4b4r07.mit-license.org"> -->
<!--         <img src="https://img.shields.io/github/license/babarot/gomi" alt="License"/> -->
<!--     </a> -->
<!--     <a href="https://github.com/babarot/gomi/releases"> -->
<!--         <img -->
<!--             src="https://img.shields.io/github/v/release/babarot/gomi" -->
<!--             alt="GitHub Releases"/> -->
<!--     </a> -->
<!--     <br /> -->
<!--     <a href="https://babarot.github.io/gomi/"> -->
<!--         <img -->
<!--             src="https://img.shields.io/website?down_color=lightgrey&down_message=donw&up_color=green&up_message=up&url=https%3A%2F%2Fbabarot.me%2Fgomi" -->
<!--             alt="Website" -->
<!--             /> -->
<!--     </a> -->
<!--     <a href="https://github.com/babarot/gomi/actions/workflows/release.yaml"> -->
<!--         <img -->
<!--             src="https://github.com/babarot/gomi/actions/workflows/release.yaml/badge.svg" -->
<!--             alt="GitHub Releases" -->
<!--             /> -->
<!--     </a> -->
<!--     <a href="https://github.com/babarot/gomi/blob/master/go.mod"> -->
<!--         <img -->
<!--             src="https://img.shields.io/github/go-mod/go-version/babarot/gomi" -->
<!--             alt="Go version" -->
<!--             /> -->
<!--     </a> -->
<!-- </p> -->

# 🗑️ Replacement for UNIX `rm` command!

<!--
- The safer alternative to the UNIX `rm` command!
- A trash can for your UNIX `rm` command!
- A smarter way to delete files with `rm`-like behavior!
- Your UNIX `rm` with a safety net!
- Undo file deletions with ease—replace `rm` with `gomi`
-->

`gomi` (meaning "trash" in Japanese) is a simple CLI tool written in Go that brings trash can functionality to the command line.

In CLI, there is no "trash" folder like in graphical file managers. So if you accidentally delete important files using the `rm` command, it can be very difficult to restore them. That's where this tool comes in. Unlike `rm`, `gomi` moves files to the trash, enabling easy restoration of important files at any time. If you're used to using `rm` in the shell, `gomi` can serve as a convenient substitute.

![demo](./docs/demo.gif)

[![Go](https://github.com/babarot/gomi/actions/workflows/build.yaml/badge.svg)](https://github.com/babarot/gomi/actions/workflows/build.yaml)
[![Release](https://github.com/babarot/gomi/actions/workflows/release.yaml/badge.svg)](https://github.com/babarot/gomi/actions/workflows/release.yaml)

## Features

- Behaves like `rm` command but not deletes actually (just moves to trash folder)
- Easy to restore with nice UI, it's super intuitive!
- Compatible with almost all flags of `rm` command
- Easy to search with fuzzy words from the history of deleted files
- Allows you to customize `gomi` behaviors/styles with YAML config file
  - specify what to show the list (regexp patterns, file globs, file size, ...)
  - specify whether colorizing the file contents or not
  - specify the command to list dir contents
  - specify the style, e.g. the color of selected files

## Usage

`gomi` is compatible with `rm` command flags (such as `-i`, `-f` and `-r`), making it easy to set up an alias like this:

```bash
alias rm=gomi
```

I developed `gomi` with the intention of replacing `rm` with another tool, so I recommend setting up the alias. However, feel free to follow your own preferences and policies. For the sake of explanation, I'll describe it assuming the alias is set.

Moves files to trash.

```bash
rm files
```

Restore the file to its original location. The `--restore` option is a bit long, so you can use the shorter version:

```bash
rm -b
```

## Installation

### From binaries

Download a command directly from [GitHub Releases][release] and drop it in your `$PATH`.

### CLI package manager

Using [afx](https://github.com/babarot/afx).

```yaml
github:
- name: babarot/gomi
  description: Trash can in CLI
  owner: babarot
  repo: gomi
  release:
    name: gomi
    tag: v1.2.0
  command:
    link:
    - from: gomi
    alias:
      rm: gomi
```
```console
afx install
```

### AUR (Arch User Repository)

Using with any AUR helpers.

```bash
yay -S gomi
```

```bash
paru -S gomi
```

https://aur.archlinux.org/packages/gomi/

## Configuration

In `gomi`, you can customize its behavior and appearance using a YAML configuration file. When you run `gomi` for the first time, a default config (like the one below) will be automatically generated at `~/.config/gomi/config.yaml`.

```yaml
core:
  restore:
    confirm: false  # if true, allows you to confirm before restoring with yes/no prompt
    verbose: true   # if true, allows you to see what was being done

ui:
  density: spacious # or compact
  preview:
    syntax_highlight: true
    colorscheme: nord  # available themes are here: https://xyproto.github.io/splash/docs/index.html
    directory_command: ls -F -A --color=always
  style:
    list_view:
      cursor: "#AD58B4"   # purple
      selected: "#5FB458" # green
      indent_on_select: false
    detail_view:
      border: "#FFFFFF"
      info_pane:
        deleted_from:
          fg: "#EEEEEE"
          bg: "#1C1C1C"
        deleted_at:
          fg: "#EEEEEE"
          bg: "#1C1C1C"
      preview_pane:
        border: "#3C3C3C"
        size:
          fg: "#EEEEDD"
          bg: "#3C3C3C"
        scroll:
          fg: "#EEEEDD"
          bg: "#3C3C3C"
  exit_message: bye!   # up-to-you
  paginator_type: dots # or arabic

inventory:
  include:
    within_days: 100 # only show files deleted within 100 days
  exclude:
    files:
    - .DS_Store      # do not show .DS_Store
    patterns:
    - "^go\\..*"     # do not show the file beginning with "go."
    globs:
    - "*.jpg"        # do not show jpeg files
    size:
      min: 0KB       # do not show empty file and dir
      max: 10GB      # do not show too big file and dir
```

## Tips

For those who come across this, the `--debug` flag provides extra debug output, though it's not included in the official help for `gomi`.

```console
gomi --debug
```

If you prefer JSON format,

```console
gomi --debug=json
```

## Related

- [andreafrancia/trash-cli](https://github.com/andreafrancia/trash-cli)
- [sindresorhus/trash](https://github.com/sindresorhus/trash)

## License

[MIT][license]

[release]: https://github.com/babarot/gomi/releases/latest
[license]: https://b4b4r07.mit-license.org
