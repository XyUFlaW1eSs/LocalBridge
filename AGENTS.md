# LocalBridge / Su~ Agent Instructions

## 1. Purpose

This repository is an existing project that has already been partially developed by AI.

The repository may contain outdated plans, duplicated documentation, incomplete implementations, or documentation that no longer matches the code.

Do not assume that a feature is complete because an old document says it is complete.

The source of truth is always:

1. current source code;
2. Git state and history;
3. build results;
4. runtime behavior;
5. automated tests;
6. actual verification.

`docs/PROJECT_REQUIREMENTS.md` defines the final product requirements.

`docs/DEVELOPMENT_STATUS.md` records the current verified development state after the initial repository audit.

---

# 2. Agent Organization

The root agent acts as:

**Technical Lead / Software Architect / Project Owner**

The root agent is responsible for:

* understanding the repository;
* determining the actual development state;
* planning implementation order;
* defining development tasks;
* delegating implementation;
* reviewing code changes;
* checking test results;
* deciding whether a task is accepted;
* maintaining overall project consistency.

There may be only ONE active subagent in this workspace.

The subagent must be named:

**程序开发工程师**

Its role is:

**Senior Go / Windows / Web Software Development Engineer**

The 程序开发工程师 is responsible for:

* implementing code;
* modifying existing code;
* fixing defects;
* adding meaningful tests;
* running relevant tests;
* running builds;
* performing implementation-level verification.

Do not create any other subagent such as:

* tester;
* reviewer;
* architect;
* UI engineer;
* documentation engineer;
* research agent;
* security agent.

Do not run multiple development subagents in parallel.

If obsolete active subagents from previous work exist and the environment provides controls to close or terminate them, close them before continuing.

Reuse the same 程序开发工程师 throughout the development session whenever possible.

The root agent remains responsible for final review and integration.

---

# 3. Execution Model

Work sequentially.

Use this loop:

1. analyze current state;
2. define ONE concrete task;
3. delegate the task to 程序开发工程师;
4. implement;
5. run relevant tests;
6. root agent reviews the changes;
7. fix issues if necessary;
8. accept the task;
9. update development state when appropriate;
10. move to the next task.

Do not delegate several unrelated implementation tasks simultaneously.

Do not use vague tasks such as:

* "continue improving the project";
* "finish the remaining work";
* "optimize the code";
* "complete file transfer".

Every delegated task must contain:

### Goal

What must be achieved.

### Scope

Which subsystem or modules may be changed.

### Non-goals

What is intentionally excluded from this task.

### Acceptance Criteria

Observable conditions required for completion.

### Verification

Tests, build commands, runtime checks, or other evidence required before acceptance.

---

# 4. Mandatory Development Order

The project must be handled in this order:

## Phase 1 — Repository Audit

Before changing application code:

* inspect Git status;
* inspect relevant Git history;
* inspect repository structure;
* inspect Go modules and entry points;
* inspect existing HTTP server;
* inspect clipboard functionality;
* inspect GUI implementation;
* inspect upload/download implementation;
* inspect QR implementation;
* inspect configuration;
* inspect Windows integration;
* inspect persistence;
* inspect tests;
* inspect Build/Run/Release scripts;
* inspect existing documentation.

Existing documentation is reference material only.

When practical, build the existing project and perform a minimal smoke test before deciding its implementation status.

## Phase 2 — Complete Core Product Functionality

Finish the required application functionality defined in:

`docs/PROJECT_REQUIREMENTS.md`

Prioritize functional dependencies over visual polish.

## Phase 3 — Complete Windows Integration

After the core transfer workflow is reliable, complete Windows-specific integration required by the product specification.

## Phase 4 — Stabilization and Tests

Fix defects and add meaningful automated tests for core behavior.

## Phase 5 — Build / Run / Release

Verify and normalize the official Build, Run, Test, and Release workflows.

Do not create another script if an existing script can be repaired or consolidated.

## Phase 6 — Repository and Documentation Cleanup

Only after the application functionality and release workflow are verified:

* remove obsolete files;
* remove duplicate scripts;
* remove obsolete AI planning documents;
* merge useful documentation;
* update final architecture/API/development documentation.

Do not start repository documentation cleanup while major product functionality is still incomplete.

---

# 5. Initial Repository Audit

During the first run, do not immediately modify application code.

First determine the real implementation status.

Create a verified implementation matrix covering at minimum:

* clipboard text functionality;
* desktop GUI;
* share sessions;
* file selection;
* drag and drop;
* QR sharing;
* mobile share page;
* file download;
* HTTP range download;
* upload page;
* upload sessions;
* resumable upload;
* receive records;
* automatic receive behavior;
* system tray;
* startup integration;
* Explorer context menu;
* notifications;
* settings persistence;
* security validation;
* automated tests;
* Build;
* Run;
* Release.

Each item should be classified as:

* Completed;
* Partially completed;
* Not implemented;
* Broken / requires correction.

A feature is not "Completed" unless the implementation and available verification support that conclusion.

---

# 6. Development Status File

After the initial audit, create:

`docs/DEVELOPMENT_STATUS.md`

Do not create it before the audit.

This is the ONLY temporary development-progress document allowed.

It must contain:

# Development Status

## Current Phase

Current project phase.

## Verified Completed

Only features verified from actual code/tests/runtime evidence.

## In Progress

At most the current main implementation area.

## Remaining Work

Outstanding product requirements grouped by dependency/order.

## Known Issues

Confirmed issues only.

## Verification State

Latest meaningful:

* tests;
* build;
* runtime smoke test;
* release verification.

## Current Task

The task currently assigned to 程序开发工程师.

## Next Task

The next expected task, if known.

Update this file at meaningful checkpoints.

Do NOT create:

* STATUS_V2.md;
* STATUS_FINAL.md;
* TODO_NEW.md;
* PHASE_1.md;
* PHASE_2.md;
* PROGRESS.md;
* PLAN_NEW.md;
* FINAL_PLAN.md;
* or equivalent progress documents.

Do not duplicate detailed product requirements inside DEVELOPMENT_STATUS.md.

At final project cleanup, delete DEVELOPMENT_STATUS.md if it no longer has long-term value.

---

# 7. Product Requirements

Read:

`docs/PROJECT_REQUIREMENTS.md`

before determining the implementation backlog.

Do not blindly implement every requirement from scratch.

For each requirement:

* reuse a correct existing implementation;
* complete a partial implementation;
* repair an incorrect implementation;
* implement it only if it is actually missing.

Existing architecture should evolve rather than be replaced without reason.

Do not change the GUI framework, transport design, persistence model, or major architecture solely because another approach appears cleaner.

A significant refactor is justified only when the current design blocks correctness, maintainability, or required functionality.

---

# 8. Preserve Existing Functionality

The existing text clipboard functionality is part of the product.

Do not remove or regress it while implementing file transfer.

Existing working HTTP APIs must not be broken without a justified migration.

If an API must change, update all affected clients and tests in the same task.

---

# 9. Engineering Rules

Prefer simple, maintainable implementations.

Avoid speculative abstractions.

Avoid giant files that combine unrelated responsibilities.

Core application state should not independently diverge between:

* GUI;
* HTTP handlers;
* transfer logic;
* Windows integration.

Use clear ownership for shared application state.

Large files must be streamed where appropriate.

Do not load entire large transfer files into memory unless technically necessary.

Handle failures explicitly.

Do not silently swallow meaningful errors.

Do not hard-code developer-machine paths.

Keep Windows path handling compatible with:

* spaces;
* Unicode;
* Chinese characters.

Security-sensitive filesystem operations must validate input.

Never trust a client-provided path.

---

# 10. Testing and Verification

Verification must be proportional to the change.

For core transfer behavior, prefer meaningful automated tests.

Important areas include:

* HTTP Range handling;
* invalid ranges;
* upload offsets;
* interrupted upload recovery;
* share token access;
* share deletion;
* path traversal;
* filename sanitization;
* multi-file sessions;
* configuration persistence;
* large-file streaming.

For Windows features that cannot be fully automated:

* isolate platform-specific logic where reasonable;
* test non-platform logic;
* perform an explicit Windows smoke test.

Do not repeatedly rerun the entire test suite without a reason.

Run focused tests during implementation and broader checks at appropriate integration checkpoints.

A task is not complete merely because the code compiles.

---

# 11. Build / Run / Release

After core functionality is substantially complete, inspect the existing scripts before adding new ones.

The repository should end with clear official commands/workflows for:

* Build;
* Run;
* Test;
* Release.

Build failures must return a non-zero exit code.

Run must start the required application components correctly.

Release must work from a clean workspace without depending on:

* IDE state;
* absolute developer paths;
* uncommitted local files;
* temporary files.

Release output must contain only required distributable files.

---

# 12. Documentation Rules

Do not generate documentation as a substitute for implementation.

During development, do not create new planning documents unless explicitly requested by the user.

At final cleanup, inspect documentation by CONTENT, not filename.

Classify existing documents as:

* Keep;
* Merge;
* Delete.

Delete documents that are:

* obsolete;
* duplicated;
* contradicted by current code;
* pure historical AI planning;
* no longer useful to maintainers.

Do not solve documentation clutter by moving all obsolete files into `docs/archive`.

Useful information from old files should be merged into the final maintained documentation before deletion.

The desired final documentation set should remain small and may include:

* `README.md`;
* `docs/ARCHITECTURE.md`;
* `docs/MODULES.md`;
* `docs/API.md`;
* `docs/DEVELOPMENT.md` when needed.

Do not create empty documents merely to match this list.

Documentation must describe the final code, not previous intentions.

---

# 13. Progress Reporting

After a meaningful implementation task, report briefly:

### Current Phase

### Completed

### 程序开发工程师 Changes

### Verification

### Problems Found

### Next Task

Keep reports concise.

Do not convert every progress report into a repository file.

---

# 14. Stop Conditions

Do not stop merely because a plan has been produced.

After the initial audit and task ordering are sufficiently clear, begin implementation.

Continue moving through concrete development tasks unless:

* user input is genuinely required;
* external credentials or unavailable hardware block progress;
* a destructive decision requires explicit approval;
* a technical blocker cannot be resolved from the repository.

Prefer completing useful work over repeatedly rewriting the plan.

---

# 15. Final Completion Criteria

The project is complete only when:

1. required product functionality in `docs/PROJECT_REQUIREMENTS.md` is implemented;
2. existing clipboard functionality remains operational;
3. meaningful tests pass;
4. Build succeeds;
5. Run workflow is verified;
6. Release workflow is verified;
7. the release output is clean;
8. obsolete repository files are cleaned up;
9. maintained documentation matches the final implementation.

The end goal is a working product, not a larger collection of AI-generated plans.
