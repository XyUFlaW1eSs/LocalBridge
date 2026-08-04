# Sprint 2.3 — LAN discovery as a reachability hint

[简体中文](sprint-2.3.zh-CN.md)

## Goal

Make trusted-LAN setup discoverable without turning unauthenticated broadcasts into access.

## Scope

- Optional UDP/IPv4 announcement and receive loop.
- Versioned, bounded discovery packet with device metadata and capabilities.
- Ephemeral discovered-peer endpoint.
- Explicit separation between discovered and paired peers.

## Non-goals

- mDNS, automatic pairing, peer authentication or outbound sync.
- Persisting discovery results.
- Guaranteeing discovery on networks that block broadcast.

## Acceptance criteria

- Discovery is disabled by default and has a configurable port/interval.
- Invalid, oversized, wrong-version and self-announcements are ignored.
- A valid announcement appears in the discovered list but not the paired list.
- Discovery packets contain no bearer or peer tokens.
- Stop cancels the listener and announcement goroutines without leaking resources.
- Manual pairing remains available when discovery is unavailable.

## Verification

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## Follow-up

The next transport slice should add authenticated peer health and outbound delivery using the
paired registry. Discovery must remain a hint even after that work lands.
