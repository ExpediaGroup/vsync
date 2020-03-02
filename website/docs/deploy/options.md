---
id: options
title: Options
sidebar_label: Options
---

We used `nomad` schedular with binary driver to deploy the job in origin and destination regions, it make easier to get destination vault tokens because of nomad vault integration.

Feel free to deploy in a way that needs minimal manual maintanence. Go Nuts!

## Artifact

There are options,

* docker image
* binary

## Securely transfer origin vault token

Its not easy to securely transfer the origin vault token to destinations.

We used destination vault for this, there could be multiple ways.

#### Step 1

Create periodic token from origin vault
``` sh
vault token create --policy vsync_origin --period 24h --orphan -display-name vsync-origin-eu-west-1-test
```

#### Step 2

Goto destination vault and under any path, say 
`secret/vsync/origin`
```
vaultToken:<created periodic vault token> 
```

#### Step 3

When you start you destination vsync app make sure you pull the origin vault token from destination vault.
