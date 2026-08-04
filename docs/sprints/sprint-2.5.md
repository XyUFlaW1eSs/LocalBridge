# Sprint 2.5 — Peer health and best-effort outbound clipboard

[简体中文](sprint-2.5.zh-CN.md)

## Goal

Exercise the paired-device and transport boundaries with real LocalBridge-to-LocalBridge
delivery while keeping durability explicitly out of scope.

## Scope

- Reusable bounded HTTP JSON transport client.
- Peer health probes using `/api/v1/system/capabilities`.
- Local clipboard event forwarding to paired peers with capability filtering.
- Remote-source suppression to prevent feedback loops.

## Non-goals

- Durable queue, retry scheduler, delivery receipts, ordering or offline replay.
- iPhone background listener or automatic iPhone push.
- TLS, automatic credential provisioning or a native client.

## Acceptance criteria

- A paired peer token can authenticate a capabilities request.
- A local clipboard event reaches a paired peer advertising `clipboard.text.push`.
- Remote clipboard events are not forwarded again.
- Unsupported or incomplete peer metadata is skipped safely.
- Request/response bounds and timeouts prevent unbounded resource use.
- Tests cover transport, event forwarding, capability filtering and lifecycle cleanup.

## Verification

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## Follow-up

Phase 3 must introduce the generic envelope, idempotent job state, durable queue, retries,
receipts and conflict policy before adding more payload types.
