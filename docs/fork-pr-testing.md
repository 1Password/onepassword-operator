# Fork PR Testing Guide

This document explains how to test external pull requests using the dispatch action workflow system.

## Overview

The testing system consists of three main workflows:

1. **E2E Tests [triggered by maintainer]** (`test-e2e.yml`) - Runs automatically for internal PRs, requires approval for external PRs
2. **Ok To Test** (`ok-to-test.yml`) - Processes slash commands to trigger fork testing
3. **E2E tests [fork]** (`test-e2e-fork.yml`) - Triggered manually via slash commands for external PRs

## How It Works

### 1. Initial PR State
When a PR is created or updated:
- The `test-e2e.yml` workflow automatically runs
- For **external PRs**: The `check-external-pr` job detects it's from a fork and **fails intentionally**
- For **internal PRs**: The workflow proceeds normally with e2e tests
- External PRs show ❌ for the check-external-pr job with a message asking for maintainer approval

### 2. Manual Approval Required
**Important**: External PRs require manual approval before workflows can run.

**Steps for maintainers:**
1. Go to the PR page
2. Click **"Approve workflow to run"** button
3. The workflow will then execute

**Note**: The `test-e2e.yml` workflow will still fail for external PRs even after approval, as it's designed to prevent automatic execution. Need to use the slash command instead.

### 3. Testing External PRs
Once the initial checks have run (and failed), maintainers can test the PR using slash commands:

#### Step-by-Step Process:

1. **Navigate to the PR**
   - Go to the pull request you want to test

2. **Add a comment with slash command**
   ```
   /ok-to-test sha=<commit-sha>
   ```
   Replace `<commit-sha>` with the actual commit SHA from the PR.
   
   **Note**: Use the short SHA.

3. **Dispatch Action Triggers**
   - The `ok-to-test.yml` workflow processes the slash command
   - It triggers a `repository_dispatch` event with type `ok-to-test-command`
   - The `test-e2e-fork.yml` workflow starts

4. **Workflow Execution**
   The fork workflow runs two jobs:
   
   **a) `e2e-tests` job** (conditional)
   - Only runs if:
     - Event is `repository_dispatch`
     - SHA parameter is not empty
     - PR head SHA contains the provided SHA
   - Calls the reusable `e2e-tests.yml` workflow
   - Runs the actual E2E tests if conditions are met
   
   **b) `update-check-status` job** (conditional)
   - Runs after `e2e-tests` completes
   - Updates the existing check for job named "e2e-tests"
   - Sets the conclusion based on `e2e-tests` result:
     - ✅ **Success** if tests pass
     - ❌ **Failure** if tests fail

## Troubleshooting

### Workflow Not Running
- **Check**: Ensure you've approved the workflow to run
- **Action**: Click "Approve workflow to run" in the PR checks tab

### SHA Not Found
- **Check**: Verify the SHA exists in the PR commits
- **Action**: Use `git log --oneline` to find valid commit SHAs

### Dispatch Not Triggering
- **Check**: Ensure the slash command format is correct
- **Action**: Use exact format: `/ok-to-test sha=<sha>`

### Check Runs Not Updating
- **Note**: The fork workflow updates the existing "E2E Tests [reusable]" check run
- **Behavior**: The same check run is updated with new results rather than creating a new one

## Security Notes

- Only users with **write** permissions can trigger dispatch commands
- The fork workflow runs in the main repository context, allowing it to update check runs
- Manual approval is required for external PR workflows
- External PRs are automatically detected and prevented from running tests automatically

## Workflow Files

- **`.github/workflows/ok-to-test.yml`** - Ok To Test - Slash command processor
- **`.github/workflows/test-e2e.yml`** - E2E Tests [triggered by maintainer] - Initial PR checks
- **`.github/workflows/test-e2e-fork.yml`** - E2E tests [fork] - Dispatch action handler
- **`.github/workflows/e2e-tests.yml`** - E2E Tests [reusable] - Reusable workflow for actual testing

## Testing Checklist

- [ ] PR created with failing check-external-pr check (for external PRs)
- [ ] Workflow approved to run (if required)
- [ ] Slash command posted with valid SHA
- [ ] Dispatch action triggered successfully
- [ ] `check-external-pr` check run updated with correct conclusion
