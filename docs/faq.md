# FAQ

## Why

### Why do we need vsync?

If you have multiple vault clusters, 1 in every region (may be under same environment).

Users need to create / update / delete secrets from each of those vaults manually for their apps (deployed in that region) to get the recent version of secret.

Instead you can ask users to update in one vault (origin) and we can propagate changes to other vaults (destinations)

This we call it sync.

Currently, vsync works only for kv v2 secrets.

### Why its named VSYNC?

Vault SYNC (oh com'on, naming is hard!!!)

Its short, easy to understand, so why not?

### Why not use as vault enterprise replication replacement? 

Vault replication is an enterprise feature with hashicorp quality.

It primarily uses streaming write ahead log to get changes propagated to other vault, which is blazing fast when compared to vsync.

### Requirements?

1. vault (atleast 1, we can use vsync in vault that is both origin and destination)
2. consul (atleast 1)
3. 50 mb of memory
4. 500 Mhz of cpu (may be less, like 300 Mhz)
5. 300 Mb of disk space (may be less)

## Deployment

### Why nomad?

* Good integration between vault and nomad
* Jobs will restart itself if something happens
* Canary deployments
* Hashicorp quality

## Failures

### If I delete the sync info in consul

Vsync will stop with a fatal error, if you restart vsync it should be fine again

### If there is no origin sync info yet for destination

Destination will wait for some time and then throw fatal error that it could not hook the consul watch on sync info

### Does not halt / stop syncing

It is designed not to stop sync because of copying one secret from origin to destination.

It should stop with fatal error for major error like if it could not start the cycle, missing required vault token permission etc