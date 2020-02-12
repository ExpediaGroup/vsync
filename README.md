# vsync

![image](./docs/vsync_text_morelight_lowres.png)

## Why

**vsync**, an easy, efficient way to sync credentials across from one origin to multiple destinations

* Parallel workers to finish the job faster
* No need of cron jobs to trigger syncing
* Cleanly closes the cycles
* Exposes telemetry data (OpenTelemetry integration in future)
* Clean vault audit log, as it uses only kv metadata for comparison
* Transform the path between origin and destination while syncing eg: secret/data/runner/stage/app1 => runnerv2/data/stage/app1/secrets without impacting apps / users
* Loopback to have origin and destination in the same vault
* Meta sync information is stored in consul
* A cleaner vault audit log as they are not clogged because of secret distribution

This does the job of replication of secrets across vaults but is not on par ( not even close to comparing ) with flexibility and performance of hashicorp's vault enterprise replication feature.

More documentation:

[How vsync Works](./docs/working.md)

[Deployment](./docs/deployment.md)

[Faq](./docs/faq.md)

## Where can I grab this tool

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
      --destination.consul.address string   destination vault address
      --destination.dc string               destination datacenter
      --destination.vault.address string    destination vault address
      --destination.vault.token string      destination vault token
  -h, --help                                help for vsync
      --log.level string                    logger level (info|debug)
      --log.type string                     logger type (console|json)
      --origin.consul.address string        origin consul address
      --origin.dc string                    origin datacenter
      --origin.vault.address string         origin vault address
      --origin.vault.token string           origin vault token
      --version                             version information

Use "vsync [command] --help" for more information about a command.
```

## Developer guide

### Local 

#### Setup

Run the script in `scripts/local_bootstrap.sh` which should create a miniature test envrionment using docker.

It creates
* 2 consuls connect via wan
  * http://localhost:6500
  * http://localhost:7500
* 2 vaults backed by respective consul
  * http://localhost:6200
  * http://localhost:7200
* unseals each vault and prints the root token which can be used in vsync config and vault ui

You can find example working configs in `configs` folder

More docs about vsync [config](./docs/deployment.md)

To create more secrets for stress test purposes, change the `seq N` in populate data section of script. populate data will use parallel to use all your cpus to create secrets faster. I have tested `N` with 10000.

#### Run

```
ORIGIN:

go run main.go origin --config ./configs/origin.json

DESTINATION:

go run main.go destination --config ./configs/dest.v2.json

> loop will have destination vault same as origin vault, useful for transforming the secret paths

go run main.go destination --config ./configs/dest_loop.v2.json
```

### TODO

* refactor how we export metrics
* more datadog & prometheus metrics
* more integration tests (hopefully with docker compose)
* change the config to use hcl format, see if its useful
* see if we can improve how we collect metedata from origin

### Good parts

* Don't change the destination metadata based on destination secrets, it is currently and it should be on origin metadata, because the destination updated time and will be different always. When we compare in next sync cycle the info will be different and we will forever be syncing

* Destination is halting
if syncmap is not present in destination vsync/ in consul, which will need manual restart to re initialize the vsync path

* For transformer regex, use https://regex101.com/. Example: https://regex101.com/r/yelNjd/1

### Required tools

### Building

### Versioning

We use git tags extensively and follow basic semantic versioning strictly with `v` prefix

`basic` semantic version -> `v{{Major}}.{{Minor}}.{{Patch}}`

`Snapshot` -> development artifact, eg. `v0.0.0-1-g1feac53`, denotes from how many commits from recent tag on same branch and current commit hash

`Release` -> public artifact, eg. `v0.0.1`

All commit messages must follow:

* Patch -> `pa: attributes: commit message`
* Minor -> `mi: attributes: commit message`
* Major -> `ma: attributes: commit message`


### Releasing

You must perform a `git tag $(NEW_VERSION)` then `git push --tags $(NEW_VERSION)`

## Contributing
Pull requests are welcome. Please refer to our [CONTRIBUTING](./CONTRIBUTING.md) file.

## Legal
This project is available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0.html).

Copyright 2019 Expedia, Inc

![gif](./docs/vsync_text_animation.gif)