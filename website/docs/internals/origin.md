---
id: origin
title: Origin
sidebar_label: Origin
---

This is the start of unidirectional connection for syncing secrets. It should point to primary vault cluster from which users expect the secrets to be propagated to other vaults in different regions.

---

## Startup

#### Step 1

Get consul and vault clients pointing to origin

#### Step 2

Check if we could read, write, update, delete in origin consul kv under sync path

#### Step 3

Check if we could read, write, update, delete in origin vault under data paths specified in config

#### Step 4

Prepare an error channel through which anyone under sync cycle can contact to throw errors

We also need to listen to error channel and check if the error at hand is fatal or not.

If not fatal, log the error with as much context available.
If fatal, stop the current sync cycle cleanly and future cycles. Log the error, inform a human, halt the program.

#### Step 5

Prepare an signal channel through which OS can send halt signals. Useful for humans to stop the whole sync program cleanly stop.

#### Step 6

A ticker is initialized for an interval (default: 1m) to start the sync cycle.
The trigger will be starting point for one cycle.

---

## Cycle

#### Step 0

A timer with timeout (default: 5m) will be created for every sync cycle. If workers get struck inbetween or something happens we do not halt vsync. Instead we wait till the timeout and kill everything created for current sync cycle. 

#### Step 1

Create a fresh `sync info` to store vsync metadata. It needs to be safe for concurrent usage.

#### Step 2

For an interval (default: 1m) we get a list of paths recursively that needs to be synced based on data paths. Example, for datapath `secret/` we get absolute paths `[secret/metadata/stage/app1, secret/metadata/stage/app2]`

#### Step 3

We create multiple worker go routines (default: 1). Each worker will generate insight and save in sync info for a given absolute path.

Each routine will be given:
* vault client pointing to origin
* shared sync info
* error channel
* multiple absolute paths but one at a time

sync info needs be safe for concurrent usage

#### Step 4

Create 1 go routine to handle saving info to consul
* if cycle is successful, save consul sync info
* if cycle has failed, abort saving info because it will corrupt existing sync info

#### Step 5

From the list of absolute paths send one path to next available worker. Once we have sent all the paths, wait for all worker go routines to complete their work.

The sender needs to be in separate routine, because we need to stop sending work to worker if we get halt signals.

#### Step 6

Reindex the sync info, for generating index info for each bucket.

#### Step 7

If everything is successful, send save signal for saving info ( index and buckets ) to consul.

If the cycle is aborted by signal, do not send the save signal for saving.

We need to cleanly close the cycle. Log appropriate cycle messages.
