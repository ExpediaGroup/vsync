---
id: general
title: General
sidebar_label: General
---

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
