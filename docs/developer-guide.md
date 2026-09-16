# Developer Guide

[简体中文](developer-guide.zh-CN.md)

These principles define how LocalBridge is built:

1. Keep the core stable; make features pluggable.
2. Documentation is part of the deliverable.
3. Every Sprint must produce a runnable build.
4. Design for five years, not five days.
5. Small commits, clear history.
6. Interfaces first, implementation second.
7. Minimize dependencies.
8. Modules communicate through events, not direct feature references.
9. Prefer configuration over hardcoding.
10. Local first: no cloud dependency for core workflows.
11. No demo code on the main branch.
12. Prefer the standard library whenever possible.

## Definition of Done

A change is done when code, tests, documentation and configuration examples agree; `go test
./...`, `go vet ./...` and the relevant platform build pass; and the change can be reviewed as
a focused commit.

## Sprint flow

```text
Design -> Implementation -> Review -> Verification -> Commit
```

Every Sprint records its goal, deliverables, acceptance criteria, known limitations and a
follow-up list. Avoid hiding unfinished behavior behind vague success messages.

## Planning workflow

Strategic scope and dependencies live in [Product Plan and Capability Map](product-plan.md)
and [Roadmap](roadmap.md). A feature enters implementation only after the following sequence:

```text
Idea / user problem
  -> capability and dependency check
  -> short design note or ADR
  -> public contract and threat/privacy review
  -> Sprint breakdown and acceptance tests
  -> implementation
  -> review and verification
  -> documentation, package and release evidence
```

### Definition of Ready

A Sprint is ready when it has:

- A concrete user scenario and a measurable outcome.
- Explicit scope, non-goals, dependencies and rollback/failure behavior.
- A data model and API/EventBus contract, if the change crosses a boundary.
- Security, privacy, retention and platform-lifecycle considerations.
- A test plan covering the happy path, duplicates, invalid input, unavailable peers and restart.
- English and Chinese documentation locations identified.

### Sprint record

Use this structure in `docs/sprints/`:

```markdown
# Sprint N — Title

## Goal
## User scenario
## Scope
## Non-goals
## Contract and data model
## Security and privacy impact
## Implementation tasks
## Acceptance criteria
## Verification evidence
## Known limitations
## Follow-up work
```

### Review checklist

Reviewers should ask:

- Does this belong in the core, transport layer or a feature module?
- Can another client implement the contract without reading internal code?
- Are delivery state, idempotency, ordering, retry and cancellation defined?
- What happens after restart, network loss, duplicate delivery or unsupported capability?
- Could logs, history, previews or diagnostics expose private content?
- Are failure paths observable without logging payload bodies?
- Is the smallest runnable package and both language docs updated?

### Configuration evolution

Configuration changes must preserve the root schema contract. Additive fields with defaults may
remain in the current version when old files retain their meaning. A semantic or breaking change
requires a new schema version, an explicit in-memory migration, rollback notes and tests for every
supported source version. Never silently accept a future version or emit secrets through effective-
configuration diagnostics.

## Release process

Before tagging a release, update the roadmap status, changelog, configuration examples,
protocol docs and known limitations. Run the relevant unit/integration tests, `go vet`,
platform builds and manual LAN checks. Build from a clean output directory, verify the package
can start and expose health/diagnostic logs, then record checksums and rollback instructions.

Security-sensitive changes require an ADR and an explicit review before merging. A feature is
not complete merely because a route returns `200`; the end-to-end user outcome, failure state,
recovery behavior and documentation must be verified.
