---
id: config
title: Config
sidebar_label: Config
---

## Hierarchy of parameters

```
"--cli-params" overrides
    "VSYNC_ENV_VARS" overrides
        "{config.vars}" overrides
            "default"
```

## Config

`--config` : cli parameter to specify the location of config

`--version` : to get build version, build commit, build time information of vsync

`syncPath` : consul kv path where vsync has to store its meta data (default: "vsync/")

> Deprecated after v0.1.1, replaced by syncPath in origin and destination

`dataPaths` : array of vault paths / mounts which needs to be synced

> Deprecated after v0.0.1, replaced by mounts in origin and destination

`log.level` : level of logs that needs to be printed to output; options: info | debug (default: "info")

`log.type` : level of logs that needs to be printed to output; options: console | json (default: "console")

`numBuckets` : sync info in consul kv will have N number of buckets and 1 index, each bucket is a map of path:insight. You will need to increase it as you hit per consul kv size limit. It must be same for origin and destinations. (default: 1)

### Origin

`origin` : top level key for all origin related config parameters

`origin.dc` : origin consul datacenter. "--origin.dc" cli param

> Deprecated after v0.1.1, replaced by dc in origin.consul.dc

`origin.vault` : origin vault top level key

`origin.vault.address` : origin vault address where we need to get metadata ( vault kv metadata ). "--origin.vault.address" cli param

`origin.vault.token` : origin vault token which has permissions to read, update, write in vault mounts. "--origin.vault.token" cli param

`origin.vault.approle.path` : origin vault approle path. "--origin.vault.approle.path" cli param (use token OR approle) (default: approle)

`origin.vault.approle.role_id` : origin vault role_id from an approle which has permissions to read, update, write in vault mounts. "--origin.vault.approle.role_id" cli param (use token OR approle)

`origin.vault.approle.secret_id` : origin vault secret_id from an approle which has permissions to read, update, write in vault mounts. "--origin.vault.approle.secret_id" cli param (use token OR approle)

`origin.mounts` : array of vault paths / mounts which needs to be synced. Each value needs to end with /. Token permissions to read, update, delete are checked for each cycle.

`origin.consul.address` : origin consul address where we need to store vsync meta data ( sync info ). "--origin.consul.address" cli param

`origin.consul.dc` : origin consul datacenter. "--origin.consul.dc" cli param

`origin.numWorkers` : number of get insights worker (default: 1)

`origin.tick` : interval for timer to start origin sync cycles. String format like 10m, 5s (default: "1m")

`origin.timout` : time limit trigger of a bomb, killing an existing sync cycle. String format like 10m, 5s (default: "5m")

`origin.renewToken` : renews origin vault periodic token and making it infinite token (default: true). See securely transfer origin vault token for more info.

### Destination

`destination` : top level key for all destination related config parameters

`destination.dc` : destination consul datacenter

> Deprecated after v0.1.1, replaced by dc in destination.consul.dc

`destination.vault` : destination vault top level key

`destination.vault.address` : destination vault address where we need to get metadata ( vault kv metadata ). "--destination.vault.address" cli param

`destination.vault.token` : destination vault token which has permissions to read, update, write in vault mounts. "--destination.vault.token" cli param

`destination.vault.approle.path` : destination vault approle path. "--destination.vault.approle.path" cli param (use token OR approle) (default: approle)

`destination.vault.approle.role_id` : destination vault role_id from an approle which has permissions to read, update, write in vault mounts. "--destination.vault.approle.role_id" cli param (use token OR approle)

`destination.vault.approle.secret_id` : destination vault secret_id from an approle which has permissions to read, update, write in vault mounts. "--destination.vault.approle.secret_id" cli param (use token OR approle)

`destination.mounts` : array of vault paths / mounts which needs to be synced. Each value needs to end with /. Token permissions to read, update, delete are checked for each cycle.

`destination.consul.dc` : destination consul datacenter.  "--destination.consul.dc" cli param

`destination.consul.address` : destination consul address where we need to store vsync meta data ( sync info ). "--destination.consul.address" cli param

`destination.numWorkers` : number of fetch and save worker (default: 1).

`destination.tick` : interval for timer to start destination sync cycles. String format like 10m, 5s (default: "1m")

`destination.timout` : time limit trigger of a bomb, killing an existing sync cycle. String format like 10m, 5s (default: "5m")

## Env

Setting `VSYNC_*` envrionment variables will also have effects. eg: "VSYNC_LOGLEVEL=debug"

## Config file

Supported format: json, hcl, yaml through [viper](https://github.com/spf13/viper)

## Examples

### Origin

```
{
    "log": {
        "level": "debug",
        "type": "console"
    },
    "logLevel": "info",
    "numBuckets": 19,
    "origin": {
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.MDLmK6gOVLL33bB5TkdnJPOB"
        },
        "consul": {
            "dc": "dc1",
            "address": "http://127.0.0.1:6500"
        },
        "mounts": [
            "secret/"
        ],
        "syncPath": "vsync/",
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    }
}
```

### Simple destination

```
{
    "log": {
        "level": "debug",
        "type": "console"
    },
    "numBuckets": 19,
    "origin": {
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.8Te1siHQnIoJ4k6el4pioQhz"
        },
        "consul": {
            "dc": "dc1",
            "address": "http://127.0.0.1:6500"
        },
        "mounts": [
            "secret/"
        ],
        "syncPath": "vsync/",
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    },
    "destination": {
        "vault": {
            "address": "http://127.0.0.1:7200",
            "token": "s.5LvYTJQhwyh2CvrZtUpnHeLb"
        },
        "consul": {
            "dc": "dc2",
            "address": "http://127.0.0.1:7500"
        },
        "syncPath": "vsync/",
        "numWorkers": 10,
        "tick": "10s",
        "timeout": "10s"
    }
}
```

### Destination with transformers

```
{
    "log": {
        "level": "debug",
        "type": "console"
    },
    "numBuckets": 19,
    "origin": {
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.8Te1siHQnIoJ4k6el4pioQhz"
        },
        "consul": {
            "dc": "dc1",
            "address": "http://127.0.0.1:6500"
        },
        "mounts": [
            "runner/"
        ],
        "syncPath": "vsync/",
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    },
    "destination": {
        "vault": {
            "address": "http://127.0.0.1:7200",
            "token": "s.5LvYTJQhwyh2CvrZtUpnHeLb"
        },
        "consul": {
            "dc": "dc2",
            "address": "http://127.0.0.1:7500"
        },
        "syncPath": "vsync/",
        "numWorkers": 10,
        "tick": "10s",
        "timeout": "10s",
        "transforms": [
            {
                "name": "v1->v2",
                "from": "(?P<mount>secret)/(?P<meta>((meta)?data))?/(?P<platform>runner)/(?P<env>(dev|test|stage|prod))?/?(?P<app>\\w+)?/?",
                "to": "runner/meta/env/app/secrets"
            }
        ]
    }
}
```

### Destination is same as origin

We are transforming from one mount to another

```
{
    "log": {
        "level": "debug",
        "type": "console"
    },
    "numBuckets": 19,
    "origin": {
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.MDLmK6gOVLL33bB5TkdnJPOB"
        },
        "consul": {
            "dc": "dc1",
            "address": "http://127.0.0.1:6500"
        },
        "mounts": [
            "runner/"
        ],
        "syncPath": "vsync/",
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    },
    "destination": {
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.MDLmK6gOVLL33bB5TkdnJPOB"
        },
        "consul": {
            "dc": "dc1",
            "address": "http://127.0.0.1:6500"
        },
        "syncPath": "vsync/",
        "numWorkers": 10,
        "tick": "10s",
        "timeout": "10s",
        "transforms": [
            {
                "name": "v1->v2",
                "from": "(?P<mount>secret)/(?P<meta>((meta)?data))?/(?P<platform>runner)/(?P<env>(dev|test|stage|prod))?/?(?P<app>\\w+)?/?",
                "to": "runner/meta/env/app/secrets"
            }
        ]
    }
}
```
