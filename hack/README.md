## Installation Script

Run the script locally:

```console
$ cat hack/install | bash
```

Run the script via curl (without cloning the repo):

```console
$ curl -sL https://gomi.dev/install | bash
```

This command is equivalent to:

```console
$ curl -sL https://raw.githubusercontent.com/babarot/gomi/HEAD/hack/install | bash
```

To specify the version to be downloaded and the installation location, use:

```console
$ curl -sL https://gomi.dev/install | VERSION=v1.4.3 PREFIX=~/.local/bin bash
```

| Environment Variable | Description | Default |
|----------------------|-------------|---------|
| `VERSION`            | Version to install (available versions are listed on [Releases](https://github.com/babarot/gomi/releases)) | `latest` |
| `PREFIX`             | Installation path | `~/bin` |
