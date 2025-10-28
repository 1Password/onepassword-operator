# Fork PR Testing Guide

This document explains how testing works for external pull requests from forks.

## Overview

The testing system consists of two main workflows:

1. **E2E Tests** (`test-e2e.yml`) - Runs automatically for internal PRs, need manual trigger on external PRs.
2. **Ok To Test** (`ok-to-test.yml`) - Dispatches `repository_dispatch` event when maintainer put's `/ok-to-test sha=<commit hash>` comment in the forked PR thread.

## How It Works

### 1. PR is created by maintainer:

For the PR created by maintainer `E2E Test` workflow starts automatically. The PR check will reflect the status of the job.

### 2. PR is created by external contributor:

For the PR created by external contributor `E2E Test` workflow **won't** start automatically.
Maintainer should make a sanity check of the changes and run it manually by:
1. Putting a comment `/ok-to-test sha=<latest commit hash>` in the PR thread.
2. `E2E Test` workflow starts.
3. After `E2E Test` workflow finishes, the commit with a link and workflow status will appear in the thread.
4. Maintainer can merge PR or request the changes based on the `E2E Test` results.


## Notes

- Only users with **write** permissions can trigger the `/ok-to-test` command.
- External PRs are automatically detected and prevented from running e2e tests automatically.
- Running e2e test on the external PR is optional. Maintainer can merge PR without running it. Maintainer decides whether it's needed to run an E2E test.
