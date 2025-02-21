# 🗑️ A Safer Alternative to the UNIX `rm` Command!


<a href="https://gomi.dev"><img align="left" width="96px" src="./docs/favicon/web-app-manifest-512x512.png" alt="image"/></a>

`gomi` (meaning "trash" in Japanese) is a simple CLI tool written in Go that adds trash can functionality to the command line.

In a typical CLI, there’s no "trash" folder like in graphical file managers. This means if you accidentally delete important files using the `rm` command, restoring them can be very difficult. That's where `gomi` comes in. Unlike `rm`, which permanently deletes files, `gomi` moves them to the trash, allowing you to easily restore files whenever necessary. If you’re used to `rm` in the shell, `gomi` works as a more convenient, safer alternative.

![demo](./docs/demo.gif)

<!-- [![Go](https://github.com/babarot/gomi/actions/workflows/build.yaml/badge.svg)](https://github.com/babarot/gomi/actions/workflows/build.yaml) -->
[![Release](https://github.com/babarot/gomi/actions/workflows/release.yaml/badge.svg)](https://github.com/babarot/gomi/actions/workflows/release.yaml)

## Features

- Functions like the `rm` command but moves files to the trash instead of permanently deleting them.
- Follows the [XDG Trash specification](https://specifications.freedesktop.org/trash-spec/latest/) for modern Linux desktop environments:
  - Supports `$XDG_DATA_HOME/Trash` or `~/.local/share/Trash`
  - Compatible with other applications using the XDG trash
- Simple and intuitive restoration process with a user-friendly interface.
- Compatible with most of the flags available for the `rm` command.
- Allows easy searching of deleted files using fuzzy search.
- Customizable via a YAML configuration file:
  - Filter what files to show (using regexp patterns, file globs, file size, etc.).
  - Customize file content colorization.
  - Define the command to list directory contents.
  - Customize visual styles (e.g., color of selected files).

For detailed information about gomi's architecture and design decisions, see [architecture.md](./docs/architecture.md).

## Usage

`gomi` is compatible with `rm` flags (like `-i`, `-f`, and `-r`), so you can easily replace `rm` by setting up an alias:

```bash
alias rm=gomi
```

I developed `gomi` as a safer replacement for `rm`, so setting up the alias is recommended. However, feel free to adjust to your preferences. The instructions below assume the alias is set.

Move files to the trash:

```bash
rm files
```

Restore a file to its original location. The `--restore` flag is a bit long, so you can use the shorthand `-b`:

```bash
rm -b
```

## Installation

### Getting Started in Seconds

Get started with `gomi` in just one command:

```bash
curl -fsSL https://gomi.dev/install | bash
```

To install it in a specific directory (e.g., `~/.local/bin`):

```bash
curl -fsSL https://gomi.dev/install | PREFIX=~/.local/bin bash
```

| Environment Variable | Description | Default |
|---|---|---|
| `VERSION` | Version to install (available versions are listed on [Releases](https://github.com/babarot/gomi/releases)) | `latest` |
| `PREFIX`  | Installation path | `~/bin` |


### From Prebuilt Binaries

Download the latest precompiled binary from [GitHub Releases][release] and place it in a directory included in your `$PATH`.

### Using a CLI package manager

[![Packaging status](https://repology.org/badge/vertical-allrepos/gomi.svg?columns=2)](https://repology.org/project/gomi/versions)

### Using [afx](https://github.com/babarot/afx)

"Write a YAML manifest, then run the `install` command.

```yaml
github:
- name: babarot/gomi
  description: Trash can in CLI
  owner: babarot
  repo: gomi
  release:
    name: gomi
    tag: v1.4.3
  command:
    link:
    - from: gomi
    alias:
      rm: gomi
```

```bash
afx install
```

### Using [Homebrew](https://brew.sh/)

```bash
brew install gomi
```

### Using [Scoop](https://scoop.sh/)

```bash
scoop bucket add babarot https://github.com/babarot/scoop-bucket
scoop install gomi
```

### Using [AUR](https://aur.archlinux.org/gomi.git)

You can install `gomi` using an AUR helper:

```bash
yay -S gomi
```

```bash
paru -S gomi
```

Find it [on AUR](https://aur.archlinux.org/packages/gomi/).

## Configuration

<!--
In `gomi`, you can customize its behavior and appearance using a YAML configuration file. When you run `gomi` for the first time, a default config (like the one below) will be automatically generated at `~/.config/gomi/config.yaml`.
-->

You can customize `gomi`'s behavior and appearance with a YAML configuration file. The first time you run `gomi`, a default config will be automatically generated at `~/.config/gomi/config.yaml`.

Here is an example of the default config:

```yaml
core:
  trash:
    strategy: "auto"   # or "xdg" or "legacy"
                       # Strategy determines which trash specification to use.

    gomi_dir: ~/.gomi  # Path to store trashed files. Can be changed to another location.
                       # Supports environment variable expansion like $HOME or ~.
                       # If empty, defaults to ~/.gomi.
                       # This config is only available on "legacy", "auto" trash strategy
  restore:
    confirm: false     # If true, prompts for confirmation before restoring (yes/no)
    verbose: true      # If true, displays detailed restoration information

  permanent_delete:
    enable: false      # If true, enables permanent deletion of files from trash.
                       # When enabled, files can be deleted permanently using the 'D' key.
                       # This operation is irreversible and bypasses the trash.
                       # Default is false for safety.
  logging:
    enabled: false     # Enable/disable logging
    level: info        # Available levels: debug, info, warn, error
    rotation:
      max_size: 10MB   # Maximum size of each log file
      max_files: 3     # Number of old log files to retain

ui:
  density: spacious # or compact
  preview:
    syntax_highlight: true
    colorscheme: nord  # Available themes: https://xyproto.github.io/splash/docs/index.html
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
    deletion_dialog: "#FF007F" # pink
  exit_message: bye!   # Customizable exit message
  paginator_type: dots # or arabic

history:
  include:
    within_days: 100 # Only show files deleted in the last 100 days
  exclude:
    files:
    - .DS_Store      # Exclude .DS_Store files
    patterns:
    - "^go\\..*"     # Exclude files starting with "go."
    globs:
    - "*.jpg"        # Exclude JPEG files
    size:
      min: 0KB       # Exclude empty files
      max: 10GB      # Exclude files larger than 10GB

```

## Debugging

Gain deeper insights into `gomi`'s operations by using the `--debug` flag:

```bash
# Show existing log file content
gomi --debug

# Follow new log entries in real-time (requires logging enabled)
gomi --debug=live
```

The behavior of the debug feature can be configured in `~/.config/gomi/config.yaml`:

```yaml
core:
  logging:
    enabled: true # Enable logging functionality
```

The `--debug` flag has two modes:

- Full-view mode (without `=live`, or with `=full`)
  - Shows the entire content of the existing log file
- Live-view mode (with `=live`)
  - Follows and displays new log entries in real-time, similar to `tail -f`.
  - While running `gomi --debug=live`, you can open another terminal and execute `gomi` commands to monitor live log updates. This feature is invaluable for troubleshooting and tracking `gomi`'s actions in real-time.

> [!NOTE]
> To use any debug features, logging must be enabled in the configuration file. The `--debug` flag only displays logs; it does not enable logging by itself.

## Related

- [andreafrancia/trash-cli](https://github.com/andreafrancia/trash-cli)
- [sindresorhus/trash](https://github.com/sindresorhus/trash)

## License

[MIT][license]

[release]: https://github.com/babarot/gomi/releases/latest
[license]: https://b4b4r07.mit-license.org
