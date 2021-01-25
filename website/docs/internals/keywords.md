---
id: keywords
title: Keywords
sidebar_label: Keywords
---

## Vault

A hashicorp product to handle secrets. [vault homepage](https://www.vaultproject.io/)

### Path

The location where the values are stored

*For kv mounts*
```
Path = mount/env/app1 [string]
        Key [string] : Value [string / json]
```

This can be confusing as `mount/env/app` is also called as `key`. So to reduce confusion we call the location a `path`

*For approle mounts*
```
Path = auth/approle/role/role_name [string]
        token_policy [string] : value [string array]
        .
        .
```

## KV store

Key value is a mount in vault which can be used to store

### KV V1

Vault mount only stores one version of key and value pair. If user wants to update, he/she will change the existing version without backup

### KV V2

Vault mount stores multiple versions *(default: 10)*. If user wants to update, hq/she will create a new version from current version, so there exists a backup inherently.

*Data*

This path contains actual data for kv. Example `mount/data/env/app1`

*Metadata*

This path contains only meta data about each kv path. Example `mount/metadata/env/app1`

If we access metadata, we could easily get a clean audit log.

## Origin

This is the source of truth, is user updates any kv pair here it should be propagated to other vaults that are in sync

There could be only `1` origin

## Destination

This is the destination where the sync must reflect the origin kv store

There could be 0 or more destinations

## Mounts

> Mounts replaced Data Paths

A list of vault mount paths which needs to be synced. It could be different between origin and destination. Vault token provided for vsync needs to have approriate permission on these paths.

```
Origin:
"mounts": [
    "secret/"
],

Destination:
"mounts": [
    "new_mount/"
    "secret/"
],
```
In future: we could have exclude paths regex and can be used to NOT sync

## Data paths

> Deprecated after v0.0.1, renamed as mounts

## Sync Info

Vsync uses consul kv to store meta data about sync.

The datastructure is designed to handle any number of entries [keys, policies] for syncing between vaults.
On the positive side it overcomes the size limit of consul kv storage as well as consul event size.

Sync Info is a collection of number of buckets (default: 20 buckets) along with 1 index.

This structure needs to be safe for concurrent usage because more workers will update their secrets at the same time.

### Insight

Each secret insight has these meta information

*struct*
```
version          -> vault kv version
updateTime       -> vault kv update time
type             -> kvV1 / kvV2 / policy
```

*eg*
```
{"version":1,"updateTime":"2019-05-14T23:41:52.904927369Z","type":"kvV2"}
```

### Bucket

Each bucket is a map with absolute path as key and insight datastructure as value

*eg*
```
"mount/data/platform/env/app1":{"version":1,"updateTime":"2019-05-14T23:41:52.904927369Z","type":"kvV2"}
,
"mount/data/platform/env/app2":{"version":1,"updateTime":"2019-05-14T23:41:52.736990492Z","type":"kvV2"}
```

### Index

An array of hashes with length as number of buckets. Each hash is constructed from contents in a particular bucket

```
["6cdb282cb3c9f6d8d3bc1d5eab88d60b728e69249f86e317c3b0d5458993bc80", ... 19 more sha256

```

> Some types are not yet implemented like kvV1 and policy

## Sync Path

A consul path to store the meta data used by vsync [sync info]

## Task

Task is a datastructure given to fetch and save worker in destination cycles

Each task has all the information needed for the worker

*struct*
```
path         -> absolute path
op           -> operation add/update/delete
insight      -> insight
```

## Transformer

Each secret path is passed through a set of transformers one by one and at last the origin secret path may be transformed to destination secret path.

To perform these changes
```
secret/data/runner/stage/app1 => runnerv2/data/stage/app1/secrets
```

*struct*
```
name (string) -> useful in logs
from (regex)  -> checked with secret path for matching
to (string)   -> could use group names present in `from` regex
```

*eg*
```
{
    "name": "v1->v2",
    "from": "(?P<mount>secret)/(?P<meta>((meta)?data))?/(?P<platform>runner)/(?P<env>(dev|test|stage|prod))?/?(?P<app>\\w+)?/?",
    "to": "runner2/meta/env/app/secrets"
}
```

## Cycle

A set of actions performed after an interval. Origin Cycle and Destination Cycle are different.

If there is a fatal failure, we abort the current cycle cleanly and cancel all future cycles then halt.

If there is a non fatal failure, we surface the error with approriate context but do not kill the cycle and future cycles.