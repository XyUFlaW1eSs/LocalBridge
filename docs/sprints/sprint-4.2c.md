# Sprint 4.2C — Receive approval workflow

[简体中文](sprint-4.2c.zh-CN.md)

## Status

Delivered. The remaining Sprint 4.2 item is native hosted-window close-to-tray behavior.

## Goal

Make the `auto_accept` setting enforce a real safety boundary without breaking resumable mobile
uploads. A phone submits bounded metadata first; file bytes are accepted only after policy permits it.

## Delivered contract

- Uploads have persisted `pending`, `active`, `completed`, `failed` and `rejected` states.
- With `auto_accept=false`, upload creation returns HTTP `202` and `pending`. The public mobile page
  polls its capability-protected status URL and waits without sending file bytes.
- `POST /api/v1/files/uploads/{upload_id}/approve` and `/reject` require loopback access or an
  authenticated management request. Approval enables the existing resumable `Content-Range` flow.
- Receive History shows pending requests with explicit Allow and Reject controls.
- A metadata-only `file.receive_requested` event produces a Windows tray notification. It contains
  only the upload ID and declared size, never a capability token, path or file content.
- Idempotent upload-creation retries return the same record and do not publish duplicate requests.

## Security and recovery

Pending uploads reserve quota but create no partial data file. Chunk writes before approval return
`409`; rejected requests stay terminal and visible to the sending page. Pending and active records
expire under the configured upload TTL and survive process restarts through the existing state store.

## Verification

Automated tests cover pending creation, duplicate event suppression, pre-approval write rejection,
remote unauthenticated approval rejection, local approval and completion, and explicit rejection.
The full Go test/vet suite, frontend syntax check, Windows release build and Linux compile gate pass.

