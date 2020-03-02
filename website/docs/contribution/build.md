---
id: build
title: Build
sidebar_label: Build
---

## Local 

### Setup

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

More docs about vsync [config](../deploy/config.md)

To create more secrets for stress test purposes, change the `seq N` in populate data section of script. populate data will use parallel to use all your cpus to create secrets faster. I have tested `N` with 10000.

### Run

ORIGIN:

```sh
go run main.go origin --config ./configs/origin.json
```

DESTINATION:

```sh
go run main.go destination --config ./configs/dest.v2.json
```

> loop will have destination vault same as origin vault, useful for transforming the secret paths

```sh
go run main.go destination --config ./configs/dest_loop.v2.json
```

---

### Versioning

We use git tags extensively and follow basic semantic versioning strictly with `v` prefix

`basic` semantic version -> `v{{Major}}.{{Minor}}.{{Patch}}`

`Snapshot` -> development artifact, eg. `v0.0.0-1-g1feac53`, denotes from how many commits from recent tag on same branch and current commit hash

`Release` -> public artifact, eg. `v0.0.1`

All commit messages must follow:

* Patch -> `pa: attributes: commit message`
* Minor -> `mi: attributes: commit message`
* Major -> `ma: attributes: commit message`

## Contributing
Pull requests are welcome. Please refer to our [CONTRIBUTING](./CONTRIBUTING.md) file.

## Legal
This project is available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0.html).