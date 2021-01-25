# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.1.0 - Jan 25 2021 
### Removed [BREAKING CHANGE]
- Assumption that first word in datapath is the mount like `secret/` but not the case in real world.
- Removed functions in vault.path.go and corresponding tests as they were having assumption about first word as mount.
### Added [FIX for BREAKING CHANGE]
- `mounts` key in origin and destination configs, its a list of mounts which needs to be synced. Take a look at [config](./website/docs/getstarted/config.md) and [examples](./configs/origin.json) to get an idea

## v0.0.1 - Feb 18 2020 
### Added
- Initial release.
