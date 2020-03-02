---
id: ci
title: CI
sidebar_label: CI
---

We are currently using github actions to produce build, test, integration-test ( yet to do ), release.


### Releasing

You must perform a `git tag $(NEW_VERSION)` then `git push --tags $(NEW_VERSION)`

Github action gets triggered only for tagged git references.

### Changelog

Currently, we have goreleaser which helps us in creating releases along with git commits that might explain what are the things in each release. We should also have a changelog.md and follow the ritual.