---
id: why
title: Why
sidebar_label: Why
---

**vsync**, an easy, efficient way to sync credentials across from one origin to multiple destinations

Developers might have their apps in multiple datacenters, each having its own vault. Its difficult for developers to update secrets in each datacenter for their apps to pickup updated secrets like database passwords. Instead we can have single origin vault, where developer will update and we can replicate the secrets to other vaults. This is where vsync fits in.

* Parallel workers to finish the job faster
* No need of cron jobs to trigger syncing
* Cleanly closes the cycles
* Exposes telemetry data (OpenTelemetry integration in future)
* Clean vault audit log, as it uses only kv metadata for comparison and they are not clogged because of secret distribution
* Transform the path between origin and destination while syncing eg: secret/data/runner/stage/app1 => runnerv2/data/stage/app1/secrets without impacting apps / users
* Loopback to have origin and destination in the same vault
* Meta sync information is stored in consul

## Similar products

### Shell scripts
Generally people comeup with shell scripts and a cron job that does the job of sequentially copying secrets from one vault to another.

### Custom application for copying the secrets
Vsync is one of them. One major missing feature is parallelly copying which does should not stop the job while copying the a particular bad secret

### Vault Enterprise
Vault Enterprise is a paid version of vault. It uses write ahead log streaming to sync blazingly fast. We all know hashicorp products are more robust. One missing piece is tranforming the paths while syncing. Its useful while performing a major platform migration without impacting any application / teams.

## Prerequiste

* All vault kv mount must be of type `KV V2`
* Currently, only works for secrets in KV mount, does not work for policies
* Currently, works only with consul as kv backend
* Single origin where developers will update their secrets
* All secrets are synced, in order to have region / environment specific secrets we may need to use secret paths like `plaform/stage/us-east-1/myapp/secrets`

## Legal
This project is available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0.html).