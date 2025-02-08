## Installation script

Run script from local:

```console
$ cat hack/install | bash
```

Run script via curl (when not cloning repo):

```console
$ curl -sL https://raw.githubusercontent.com/babarot/gomi/HEAD/hack/install | GOMI_VERSION=v1.2.2 bash
```

env | description | default
---|---|---
`GOMI_VERSION` | gomi version, available versions are on [releases](https://github.com/babarot/gomi/releases) | `latest` 
`GOMI_BIN_DIR` | Path to install | `~/bin` 
