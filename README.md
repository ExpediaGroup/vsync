# vsync

vsync, an easy, efficient way to sync credentials across from one origin to multiple destinations

Developers might have their apps in multiple datacenters, each having its own vault. Its difficult for developers to update secrets in each datacenter for their apps to pickup updated secrets like database passwords. Instead we can have single origin vault, where developer will update and we can replicate the secrets to other vaults. This is where vsync fits in.

* Parallel workers to finish the job faster
* No need of cron jobs to trigger syncing
Cleanly closes the cycles
* Exposes telemetry data (OpenTelemetry integration in future)
* Clean vault audit log, as it uses only kv metadata for comparison and they are not clogged because of secret distribution
* Transform the path between origin and destination while syncing eg: secret/data/runner/stage/app1 => runnerv2/data/stage/app1/secrets without impacting apps / users
* Loopback to have origin and destination in the same vault
* Meta sync information is stored in consul

> NOTE: Not prod ready yet, esp because of tests and telemetry

Website: https://expediagroup.github.io/vsync