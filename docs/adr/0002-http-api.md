# ADR 0002: Standard library HTTP and JSON

[简体中文](0002-http-api.zh-CN.md)

## Decision

Use `net/http` with Go's method-aware `ServeMux` and JSON payloads under `/api/v1`.

## Rationale

The protocol is small, debuggable from a browser or Shortcut, and avoids a framework dependency.
The versioned path gives future clients a stable compatibility boundary.
