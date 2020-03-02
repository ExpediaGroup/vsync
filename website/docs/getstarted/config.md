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

`dataPaths` : array of vault paths / mounts which needs to be synced

`log.level` : level of logs that needs to be printed to output; options: info | debug (default: "info")

`log.type` : level of logs that needs to be printed to output; options: console | json (default: "console")

`numBuckets` : sync info in consul kv will have N number of buckets and 1 index, each bucket is a map of path:insight. You will need to increase it as you hit per consul kv size limit. It must be same for origin and destinations. (default: 1)

### Origin

`origin` : top level key for all origin related config parameters

`origin.dc` : origin consul datacenter. "--origin.dc" cli param

`origin.vault` : origin vault top level key

`origin.vault.address` : origin vault address where we need to get metadata ( vault kv metadata ). "--origin.vault.address" cli param

`origin.vault.token` : origin vault token which has permissions to read, update, write in vault datapaths. "--origin.vault.token" cli param

`origin.consul.address` : origin consul address where we need to store vsync meta data ( sync info ). "--origin.consul.address" cli param

`origin.numWorkers` : number of get insights worker (default: 1)

`origin.tick` : interval for timer to start origin sync cycles. String format like 10m, 5s (default: "1m")

`origin.timout` : time limit trigger of a bomb, killing an existing sync cycle. String format like 10m, 5s (default: "5m")

`origin.renewToken` : renews origin vault periodic token and making it infinite token (default: true). See securely transfer origin vault token for more info.

### Destination

`destination` : top level key for all destination related config parameters

`destination.dc` : destination consul datacenter

`destination.vault` : destination vault top level key

`destination.vault.address` : destination vault address where we need to get metadata ( vault kv metadata ). "--destination.vault.address" cli param

`destination.vault.token` : destination vault token which has permissions to read, update, write in vault datapaths. "--destination.vault.token" cli param

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
    "syncPath": "vsync/",
    "dataPaths": [
        "secret/"
    ],
    "log": {
        "level": "debug",
        "type": "console"
    },
    "logLevel": "info",
    "numBuckets": 19,
    "origin": {
        "dc": "dc1",
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.MDLmK6gOVLL33bB5TkdnJPOB"
        },
        "consul": {
            "address": "http://127.0.0.1:6500"
        },
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    }
}
```

### Simple destination

```
{
    "syncPath": "vsync/",
    "dataPaths": [
        "secret/"
    ],
    "log": {
        "level": "debug",
        "type": "console"
    },
    "numBuckets": 19,
    "origin": {
        "dc": "dc1",
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.8Te1siHQnIoJ4k6el4pioQhz"
        },
        "consul": {
            "address": "http://127.0.0.1:6500"
        },
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    },
    "destination": {
        "dc": "dc2",
        "vault": {
            "address": "http://127.0.0.1:7200",
            "token": "s.5LvYTJQhwyh2CvrZtUpnHeLb"
        },
        "consul": {
            "address": "http://127.0.0.1:7500"
        },
        "numWorkers": 10,
        "tick": "10s",
        "timeout": "10s"
    }
}
```

### Destination with transformers

```
{
    "syncPath": "vsync/",
    "dataPaths": [
        "runner/"
    ],
    "log": {
        "level": "debug",
        "type": "console"
    },
    "numBuckets": 19,
    "origin": {
        "dc": "dc1",
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.8Te1siHQnIoJ4k6el4pioQhz"
        },
        "consul": {
            "address": "http://127.0.0.1:6500"
        },
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    },
    "destination": {
        "dc": "dc2",
        "vault": {
            "address": "http://127.0.0.1:7200",
            "token": "s.5LvYTJQhwyh2CvrZtUpnHeLb"
        },
        "consul": {
            "address": "http://127.0.0.1:7500"
        },
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
    "syncPath": "vsync/",
    "dataPaths": [
        "runner/"
    ],
    "log": {
        "level": "debug",
        "type": "console"
    },
    "numBuckets": 19,
    "origin": {
        "dc": "dc1",
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.MDLmK6gOVLL33bB5TkdnJPOB"
        },
        "consul": {
            "address": "http://127.0.0.1:6500"
        },
        "numWorkers": 5,
        "tick": "10s",
        "timeout": "10s"
    },
    "destination": {
        "dc": "dc1",
        "vault": {
            "address": "http://127.0.0.1:6200",
            "token": "s.MDLmK6gOVLL33bB5TkdnJPOB"
        },
        "consul": {
            "address": "http://127.0.0.1:6500"
        },
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
