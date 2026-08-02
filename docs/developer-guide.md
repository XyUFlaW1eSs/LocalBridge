# Developer Guide

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
