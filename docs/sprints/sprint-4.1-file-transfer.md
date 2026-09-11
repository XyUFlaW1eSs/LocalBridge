# Sprint 4.1 — File Share and Resumable Transfer Foundation

[简体中文](sprint-4.1-file-transfer.zh-CN.md)

Status: assigned to the `程序开发` child task; not accepted yet.

## Goal

Replace the assumption that iPhone can paste arbitrary files with a safe, URL/QR-based,
multi-file transfer workflow. This sprint delivers the backend contract and a minimal mobile
page boundary. It does not claim to deliver the Windows native shell, tray or Explorer menu.

## Scope

- A persisted share batch containing one or more files selected on the Windows host.
- Opaque public share token, expiry, revocation and bounded file metadata.
- JSON APIs to create, inspect, list and revoke shares.
- Public mobile HTML page that lists the files and exposes download links.
- HTTP Range download with correct `206`/`Content-Range` behavior and checksum metadata.
- Upload session API using sequential `Content-Range` chunks, committed-offset status and resume.
- Atomic completion, whole-file SHA-256 verification, safe receive directory and history records.
- Tests for multi-file grouping, invalid paths, expiry/revocation, ranges, interrupted upload,
  duplicate chunks, checksum failure and restart recovery.
- English/Chinese protocol, configuration, deployment and sprint documentation.

## Non-goals

- Native Windows GUI, tray icon, startup registration or Explorer shell extension.
- Notification sounds or OS notification integration.
- URL push, image clipboard and general plugin APIs.
- Silent acceptance of uploads by default.

## Acceptance

1. A test creates one share from multiple files and receives one share record with expandable
   file metadata and no unrestricted source path in the public response.
2. A public URL can list and download each file; a resumed Range request returns the exact bytes.
3. An upload can stop after an arbitrary chunk, restart, query its offset and continue to a
   verified final file without corrupting an existing destination.
4. Expired or revoked tokens cannot read, upload or mutate transfer state.
5. Traversal, symlink escape, oversized body, invalid Range and checksum mismatch are rejected.
6. `go test ./...`, `go vet ./...`, `go build ./cmd/localbridge` and `git diff --check` pass.

