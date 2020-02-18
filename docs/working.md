# How vsync works

## Keywords

### Vault

A hashicorp product to handle secrets. [vault homepage](https://www.vaultproject.io/)

#### Path

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

### KV store

Key value is a mount in vault which can be used to store

#### KV V1

Vault mount only stores one version of key and value pair. If user wants to update, he/she will change the existing version without backup

#### KV V2

Vault mount stores multiple versions *(default: 10)*. If user wants to update, hq/she will create a new version from current version, so there exists a backup inherently.

*Data*

This path contains actual data for kv. Example `mount/data/env/app1`

*Metadata*

This path contains only meta data about each kv path. Example `mount/metadata/env/app1`

If we access metadata, we could easily get a clean audit log.

### Origin

This is the source of truth, is user updates any kv pair here it should be propagated to other vaults that are in sync

There could be only `1` origin

### Destination

This is the destination where the sync must reflect the origin kv store

There could be 0 or more destinations

### Data paths

A list of vault mount paths which needs to be synced. It could be different between origin and destination. Vault token provided for vsync needs to have approriate permission on these paths.

```
Origin:
"dataPaths": [
    "secret/"
],

Destination:
"dataPaths": [
    "new_mount/"
    "secret/"
],
```
In future: we could have exclude paths regex and can be used to NOT sync

### Sync Info

Vsync uses consul kv to store meta data about sync.

The datastructure is designed to handle any number of entries [keys, policies] for syncing between vaults.
On the positive side it overcomes the size limit of consul kv storage as well as consul event size.

Sync Info is a collection of number of buckets (default: 20 buckets) along with 1 index.

This structure needs to be safe for concurrent usage because more workers will update their secrets at the same time.

#### Insight

Each secret insight has these meta information

```
version          -> vault kv version
updateTime       -> vault kv update time
type             -> kvV1 / kvV2 / policy
```

eg:

```
{"version":1,"updateTime":"2019-05-14T23:41:52.904927369Z","type":"kvV2"}
```

#### Bucket

Each bucket is a map with absolute path as key and insight datastructure as value

eg:

```
"mount/data/platform/env/app1":{"version":1,"updateTime":"2019-05-14T23:41:52.904927369Z","type":"kvV2"}
,
"mount/data/platform/env/app2":{"version":1,"updateTime":"2019-05-14T23:41:52.736990492Z","type":"kvV2"}
```

#### Index

An array of hashes with length as number of buckets. Each hash is constructed from contents in a particular bucket

```
["6cdb282cb3c9f6d8d3bc1d5eab88d60b728e69249f86e317c3b0d5458993bc80", ... 19 more sha256

```

> Some types are not yet implemented like kvV1 and policy

### Sync Path

A consul path to store the meta data used by vsync [sync info]

### Task

Task is a datastructure given to fetch and save worker in destination cycles

Each task has all the information needed for the worker
Task
```
path         -> absolute path
op           -> operation add/update/delete
insight      -> insight
```

### Transformer
```
name (string) -> useful in logs
from (regex)  -> checked with secret path for matching
to (string)   -> could use group names present in `from` regex
```

eg:

```
{
    "name": "v1->v2",
    "from": "(?P<mount>secret)/(?P<meta>((meta)?data))?/(?P<platform>runner)/(?P<env>(dev|test|stage|prod))?/?(?P<app>\\w+)?/?",
    "to": "runner2/meta/env/app/secrets"
}

secret/data/runner/stage/app1 => runnerv2/data/stage/app1/secrets
```

Each secret path is passed through a set of transformers one by one and at last the origin secret path may be transformed to destination secret path.

## Prerequiste

* All vault kv mount must be of type `KV V2`
* Currently, only works for secrets in KV mount, does not work for policies
* Vault in multiple regions, under same environment `stage` has `us-east-1`, `us-east4` vaults
* Environment in vault paths like `multipaas/stage/myapp/secrets` will be so useful to mitigate confusions

## Cycle

A set of actions performed after an interval. Origin Cycle and Destination Cycle are different.

If there is a fatal failure, we abort the current cycle cleanly and cancel all future cycles then halt.

If there is a non fatal failure, we surface the error with approriate context but do not kill the cycle and future cycles.

---

## Origin

This is the start of unidirectional connection for syncing secrets. It should point to primary vault cluster from which users expect the secrets to be propagated to other vaults in different regions.

### Startup

*Step 1*

Get consul and vault clients pointing to origin

*Step 2*

Check if we could read, write, update, delete in origin consul kv under sync path

*Step 3*

Check if we could read, write, update, delete in origin vault under data paths specified in config

*Step 4*

Prepare an error channel through which anyone under sync cycle can contact to throw errors

We also need to listen to error channel and check if the error at hand is fatal or not.

If not fatal, log the error with as much context available.
If fatal, stop the current sync cycle cleanly and future cycles. Log the error, inform a human, halt the program.

*Step 5*

Prepare an signal channel through which OS can send halt signals. Useful for humans to stop the whole sync program cleanly stop.

*Step 6*

A ticker is initialized for an interval (default: 1m) to start the sync cycle.
The trigger will be starting point for one cycle.

### Cycle

*Step 0*

A timer with timeout (default: 5m) will be created for every sync cycle. If workers get struck inbetween or something happens we do not halt vsync. Instead we wait till the timeout and kill everything created for current sync cycle. 

*Step 1*

Create a fresh `sync info` to store vsync metadata. It needs to be safe for concurrent usage.

*Step 2*

For an interval (default: 1m) we get a list of paths recursively that needs to be synced based on data paths. Example, for datapath `secret/` we get absolute paths `[secret/metadata/stage/app1, secret/metadata/stage/app2]`

*Step 3*

We create multiple worker go routines (default: 1). Each worker will generate insight and save in sync info for a given absolute path.

Each routine will be given:
* vault client pointing to origin
* shared sync info
* error channel
* multiple absolute paths but one at a time

sync info needs be safe for concurrent usage

*Step 4*

Create 1 go routine to handle saving info to consul
* if cycle is successful, save consul sync info
* if cycle has failed, abort saving info because it will corrupt existing sync info

*Step 5*

From the list of absolute paths send one path to next available worker. Once we have sent all the paths, wait for all worker go routines to complete their work.

The sender needs to be in separate routine, because we need to stop sending work to worker if we get halt signals.

*Step 6*

Reindex the sync info, for generating index info for each bucket.

*Step 7*

If everything is successful, send save signal for saving info ( index and buckets ) to consul.

If the cycle is aborted by signal, do not send the save signal for saving.

We need to cleanly close the cycle. Log appropriate cycle messages.

---

## Destination

This is the end of unidirectional connection for syncing secrets. It should point to secondary vault clusters in different regions from which regional apps can pull out secrets.

### Startup

*Step 1*

Get consul and vault clients pointing to origin

Get consul and vault clients pointing to destinations

*Step 2*

Check if we could read, write, update, delete in origin consul kv under origin and destination sync paths

*Step 3*

Check if we could read in origin vault under data paths specified in config

Check if we could read, write, update, delete in destination vault under data paths specified in config

*Step 4*

Prepare an error channel through which anyone under sync cycle can contact to throw errors

We also need to listen to error channel and check if the error at hand is fatal or not.

If not fatal, log the error with as much context available.
If fatal, stop the current sync cycle cleanly and future cycles. Log the error, inform a human, halt the program.

*Step 5*

Prepare an signal channel through which OS can send halt signals. Useful for humans to stop the whole sync program cleanly stop.

*Step 7*

Prepare a consul watch on origin sync index so whenever there is a change in consul index change we can run destination cycle

As a backup to consul watch, we also have a ticker initialized for interval (default: 1m) to run destination cycle

*Step 6*

A ticker is initialized for an interval (default: 1m) to start the sync cycle.
The trigger will be starting point for one cycle.

*Step 7*

Transformers are initialized as a stack, bottom most are from default pack, top ones are from config file.

### Cycle

*Step 0*

A timer with timeout (default: 5m) will be created for every sync cycle. If workers get struck inbetween or something happens we do not halt vsync. Instead we wait till the timeout and kill everything created for current sync cycle. 

*Step 1*

If triggered either via origin consul watch or ticker, we get origin sync info and destination sync info

*Step 2*

Compare the infos from origin and destination.

Comparison starts basics like number of buckets then moves on to do index matching.

If index of a specific bucket is different between origin and destination, then we assume that bucket's contains are not changed.

If index of a specific bucket is changed between origin and destination, then we check each key value pair in that bucket map.

Origin is the source of truth and has more priority in solving differences in comparison.

For every secret we first check version and type then updated time to make sure we update destinations based on changes from origin

> Edge Case:
User at origin has deleted the meta information, so it resets version numbers. User has also updated a new secret in same path, same number of times to match the version as old secret. In that case, the updated time will differ; hence we should be able to sync new changes

The output of comparison will be 3 lists [addTask, updateTask, deleteTask] which can be given for workers to fetch from origin and save to destination. 

We have effectively zeroed redundent copying of unaltered secrets.

*Step 3*

We create multiple worker go routines (default: 1). Each worker will fetch secret from origin and save in destination. 

Each routine will be given:
* vault client pointing to origin
* vault client pointing to destination
* pack of tranformers
* shared sync info
* error channel
* multiple origin absolute paths but one at a time

sync info needs be safe for concurrent usage

Each worker
1. will call origin vault for secret data
2. transform the path for destination if necessary
3. save the secret in destination under new path
4. copy the origin sync info for that secret to destination sync info. The reason for copying origin and not saving destination metainfo as such, is we need to compare origin and destination sync infos in future cycles

*Step 4*

Create 1 go routine to handle saving info to consul
* if cycle is successful, save consul sync info
* if cycle has failed, abort saving info because it will corrupt existing sync info

*Step 5*

From the list of absolute paths from comparison, send one path to next available worker. Once we have sent all the paths, wait for all worker go routines to complete their work.

The sender needs to be in separate routine, because we need to stop sending work to worker if we get halt signals.

*Step 6*

Reindex the destination sync info, for generating index info for each bucket.

*Step 7*

If everything is successful, send save signal for saving info ( index and buckets ) to destination consul.

If the cycle is aborted by signal, do not send the save signal for saving.

We need to cleanly close the cycle. Log appropriate cycle messages.

