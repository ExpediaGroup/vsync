---
id: install
title: Install
sidebar_label: Install
---

### Manual download

1. Goto [releases](https://github.com/ExpediaGroup/vsync/releases) page
2. Download the latest binary for your OS
3. Place the binary somewhere in global path, like `/usr/local/bin`

### Docker images
Find the latest release tag from github release page of this project

```
TODO: add docker image 
```

## Usage

```
A tool that sync secrets between different vaults probably within same environment vaults

Usage:
  vsync [flags]
  vsync [command]

Available Commands:
  destination Performs comparisons of sync data structures and copies data from origin to destination for nullifying the diffs
  help        Help about any command
  origin      Generate sync data structure in consul kv for entities that we need to distribute

Flags:
  -c, --config string                       load the config file along with path (default is $HOME/.vsync.json)
      --destination.consul.address string   destination consul address
      --destination.consul.dc string        destination consul datacenter
      --destination.vault.address string    destination vault address
      --destination.vault.token string      destination vault token
  -h, --help                                help for vsync
      --log.level string                    logger level (info|debug)
      --log.type string                     logger type (console|json)
      --origin.consul.address string        origin consul address
      --origin.consul.dc string             origin consul datacenter
      --origin.vault.address string         origin vault address
      --origin.vault.token string           origin vault token
      --version                             version information

Use "vsync [command] --help" for more information about a command.
```