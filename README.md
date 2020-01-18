<p align="center">
  <img src="./docs/screenshot.png" width="500" alt="gomi">
</p>

[![](http://img.shields.io/github/release/b4b4r07/gomi.svg?style=flat-square)][release] [![](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

`gomi` is a simple trash tool that works on CLI, written in Go

The concept of the trashcan does not exist in Command-line interface ([CLI](http://en.wikipedia.org/wiki/Command-line_interface)). If you have deleted an important file by mistake with the `rm` command, it would be difficult to restore. Then, it's this `gomi`. Unlike `rm` command, it is possible to easily restore deleted files because `gomi` have the trashcan for the CLI. It is also possible to work with trashcan for Graphical user interface ([GUI](http://en.wikipedia.org/wiki/Graphical_user_interface)).

## Features

- Like a `rm` command but not unlink (delete) in fact (just move to another place)
- Easy to restore, super intuitive
- Compatible with `rm` command, e.g. `-r`, `-f` options
- Nice UI, awesome CLI UX

## Installation

Download the binary from [GitHub Releases][release] and drop it in your `$PATH`.

- [Darwin / Mac](https://github.com/b4b4r07/gomi/releases/latest)
- [Linux](https://github.com/b4b4r07/gomi/releases/latest)

## Versus

- [andreafrancia/trash-cli](https://github.com/andreafrancia/trash-cli)
- [sindresorhus/trash](https://github.com/sindresorhus/trash)

## License

[MIT](license) (c) BABAROT (a.k.a. [b4b4r07](https://tellme.tokyo))

[release]: https://github.com/b4b4r07/gomi/releases
[license]: https://b4b4r07.mit-license.org
