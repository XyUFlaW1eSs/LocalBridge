# Engineering Workflow and Agent Responsibilities

[简体中文](engineering-workflow.zh-CN.md)

## Purpose

LocalBridge is developed as a long-lived platform. The main task owns product direction,
architecture, contracts, prioritization and acceptance. The child task named `程序开发`
owns implementation work that is explicitly assigned to it. Neither role may silently change
the product scope or claim completion without evidence.

## Roles

### Main controller

- Review the current repository before assigning work.
- Turn product goals into small, dependency-ordered task packets.
- Define API, storage, security, UX and compatibility contracts before implementation.
- Assign implementation work to `程序开发` in an isolated worktree.
- Review diffs, tests, security boundaries, documentation and user-visible behavior.
- Reject incomplete work and send focused follow-up tasks.
- Integrate only changes that pass the acceptance gate.

### `程序开发` child task

- Inspect existing code and preserve compatible behavior.
- Implement only the assigned task packet.
- Add tests for normal, failure, restart, security and recovery paths.
- Keep English and Chinese documentation synchronized.
- Run formatting, tests, vet and build before reporting completion.
- Commit each independently reviewable slice and report the commit ID and known gaps.

## Task packet format

Every implementation task should state:

1. User-visible outcome.
2. Packages, routes, storage and configuration affected.
3. Security and privacy limits.
4. Compatibility requirements with existing modules.
5. Required tests and manual acceptance steps.
6. Documentation files that must be updated.
7. Explicit non-goals for the slice.

## Review gates

An implementation slice is not delivered merely because it compiles. The main controller
checks the following in order:

```text
Contract review
    -> implementation diff review
    -> unit and HTTP integration tests
    -> go vet and build
    -> security / data-loss review
    -> manual user-flow verification
    -> bilingual docs and release notes
```

File transfer slices additionally require interrupted-transfer tests, range validation,
checksum verification, expiry/revocation tests, path traversal tests and restart recovery.
GUI slices additionally require a clean-machine launch check, drag/drop or file-picker check,
tray/close behavior, settings persistence and a usable mobile viewport.

## Worktree and commit policy

The child task works in an isolated Git worktree. Commits should be narrow and descriptive.
The main controller reviews the commit before integration; generated data, credentials, local
configuration and build artifacts must never be committed.

## Current execution order

1. File-share and resumable-transfer protocol plus durable state.
2. Mobile download/upload web pages and QR metadata.
3. Desktop web UI shell: sharing, receive history and settings.
4. Windows packaging: tray, close-to-tray, startup and Explorer context menu.
5. Notifications, sounds, transfer recovery and end-to-end tests.
6. Security review, performance testing, bilingual release docs and v1.0 gate.

