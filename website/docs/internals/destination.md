---
id: destination
title: Destination
sidebar_label: Destination
---

This is the end of unidirectional connection for syncing secrets. It should point to secondary vault clusters in different regions from which regional apps can pull out secrets.

---

## Startup

#### Step 1

Get consul and vault clients pointing to origin

Get consul and vault clients pointing to destinations

#### Step 2

Check if we could read, write, update, delete in origin consul kv under origin and destination sync paths

#### Step 3

Check if we could read in origin vault under data paths specified in config

Check if we could read, write, update, delete in destination vault under data paths specified in config

#### Step 4

Prepare an error channel through which anyone under sync cycle can contact to throw errors

We also need to listen to error channel and check if the error at hand is fatal or not.

If not fatal, log the error with as much context available.
If fatal, stop the current sync cycle cleanly and future cycles. Log the error, inform a human, halt the program.

#### Step 5

Prepare an signal channel through which OS can send halt signals. Useful for humans to stop the whole sync program cleanly stop.

#### Step 7

Prepare a consul watch on origin sync index so whenever there is a change in consul index change we can run destination cycle

As a backup to consul watch, we also have a ticker initialized for interval (default: 1m) to run destination cycle

#### Step 6

A ticker is initialized for an interval (default: 1m) to start the sync cycle.
The trigger will be starting point for one cycle.

#### Step 7

Transformers are initialized as a stack, bottom most are from default pack, top ones are from config file.

## Cycle

#### Step 0

A timer with timeout (default: 5m) will be created for every sync cycle. If workers get struck inbetween or something happens we do not halt vsync. Instead we wait till the timeout and kill everything created for current sync cycle. 

#### Step 1

If triggered either via origin consul watch or ticker, we get origin sync info and destination sync info

#### Step 2

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

#### Step 3

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

#### Step 4

Create 1 go routine to handle saving info to consul
* if cycle is successful, save consul sync info
* if cycle has failed, abort saving info because it will corrupt existing sync info

#### Step 5

From the list of absolute paths from comparison, send one path to next available worker. Once we have sent all the paths, wait for all worker go routines to complete their work.

The sender needs to be in separate routine, because we need to stop sending work to worker if we get halt signals.

#### Step 6

Reindex the destination sync info, for generating index info for each bucket.

#### Step 7

If everything is successful, send save signal for saving info ( index and buckets ) to destination consul.

If the cycle is aborted by signal, do not send the save signal for saving.

We need to cleanly close the cycle. Log appropriate cycle messages.