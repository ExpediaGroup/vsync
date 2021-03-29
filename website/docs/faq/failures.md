---
id: failures
title: Failures
sidebar_label: Failures
---

## Failures

### If I delete the sync info in consul

Vsync will stop with a fatal error, if you restart vsync it should be fine again

### If there is no origin sync info yet for destination

Destination will wait for some time and then throw fatal error that it could not hook the consul watch on sync info

### Does not halt / stop syncing

It is designed not to stop sync because of copying one secret from origin to destination.

It should stop with fatal error for major error like if it could not start the cycle, missing required vault token permission etc

### Sync index not found

Vsync Origin should be started and running successfully before Vsync Destination

You will most probably get `cannot get sync info from origin consul`
