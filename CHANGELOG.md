# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.3.0 - Dec 15 2021
### Add

- adding `ignoreDeletes` boolean flag for stopping vsync destination from deleting all secrets at once and then recreating them, some times its scary even though its a soft delete ( deleting latest version )
- This issue is caused when vsync origin uploads empty sync info in case there is some issue with origin vault.
- Once a delete is ignored, its also not stored in destination sync info so that we can pull up the changes easily between origin and destination

## v0.2.1 - Aug 16 2021
### Updated

- A cleaner way we check permissions of vault token on a path by using sys/capabilities-self api call to get a list of capabilities


## v0.2.0 - May 27 2021
### Updated [BREAKING CHANGE]

- Sync Path were initially on the config root, its moved to origin and destination separately so that we can have different paths to store sync data even with same consul
- `syncPath` -> `origin.syncPath` & `destination.syncPath`
- Variable `origin.dc` and `destination.dc` was ambiguous, so moved to `origin.consul.dc` & `destination.consul.dc`

## v0.1.1 - Mar 30 2021
### Added
- Approle support for getting vault token.

## v0.1.0 - Jan 25 2021 
### Removed [BREAKING CHANGE]
- Assumption that first word in datapath is the mount like `secret/` but not the case in real world.
- Removed functions in vault.path.go and corresponding tests as they were having assumption about first word as mount.
### Added [FIX for BREAKING CHANGE]
- `mounts` key in origin and destination configs, its a list of mounts which needs to be synced. Take a look at [config](./website/docs/getstarted/config.md) and [examples](./configs/origin.json) to get an idea

## v0.0.1 - Feb 18 2020 
### Added
- Initial release.
