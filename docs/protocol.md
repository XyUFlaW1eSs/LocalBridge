# Protocol and API

[简体中文](protocol.zh-CN.md)

For the complete clipboard field reference, see [Clipboard Module and API](clipboard.md).

Base URL: `http://<windows-ip>:8899`.

Phase 1 transports UTF-8 text only. JSON requests and responses use UTF-8. Times are RFC
3339 UTC strings.

## Health

`GET /api/v1/system/health`

```json
{"status":"ok","service":"localbridge","time":"2026-08-02T00:00:00Z"}
```

## Push clipboard

`POST /api/v1/clipboard`

The preferred body is a JSON object:

```json
{
  "id": "iphone-personal-1710000000000000000",
  "type": "text",
  "mime_type": "text/plain",
  "content": "hello from iPhone",
  "hash": "<lowercase-sha256-of-content>",
  "device_id": "iphone-personal"
}
```

| Field | Type | Required | Default / meaning |
|---|---|---:|---|
| `content` | string | Yes* | UTF-8 clipboard text; empty content is rejected. |
| `text` | string | No | Compatibility alias used when `content` is missing or empty. |
| `type` | string | No | Defaults to `text`; Phase 1 is text-only. |
| `mime_type` | string | No | Defaults to `text/plain`. |
| `device_id` | string | No | Defaults to server `device.id`; identifies the producing device. |
| `id` | string | No | Generated from device ID and Unix nanoseconds when omitted. |
| `hash` | string | No | Lowercase SHA-256 deduplication key; computed when omitted. |

\* A non-empty `text` alias can satisfy the content requirement.

Raw UTF-8 text is also accepted with `Content-Type: text/plain`; it is interpreted as
`content` with `type: text` and `mime_type: text/plain`.

The default maximum is `clipboard.max_text_bytes: 1048576` bytes. The limit counts UTF-8
bytes. A successful response is HTTP `200`:

```json
{
  "accepted": true,
  "item": {
    "id": "iphone-personal-1710000000000000000",
    "type": "text",
    "mime_type": "text/plain",
    "content": "hello from iPhone",
    "hash": "<lowercase-sha256>",
    "device_id": "iphone-personal",
    "source": "remote",
    "created_at": "2026-08-02T00:00:00Z"
  }
}
```

`accepted: false` means the hash matched the current latest item, so the Windows clipboard
was not written again. The `item` fields are `id`, `type`, `mime_type`, `content`, `hash`,
`device_id`, `source` (`remote` or `local`) and `created_at`.

## Pull clipboard

`GET /api/v1/clipboard/latest`

Returns the latest `item` directly:

```json
{
  "id": "windows-pc-1710000000000000000",
  "type": "text",
  "mime_type": "text/plain",
  "content": "copied on Windows",
  "hash": "<lowercase-sha256>",
  "device_id": "windows-pc",
  "source": "local",
  "created_at": "2026-08-02T00:00:00Z"
}
```

Before any item exists, it returns HTTP `404` and `{"error":"clipboard is empty"}`.

## Windows local clipboard direction

When text is copied directly on Windows, the watcher reads it and updates the in-memory
`latest` item. LocalBridge does not currently know a peer iPhone endpoint and does not make
an automatic HTTP request to the phone. The iPhone must run the Pull Shortcut to GET
`latest` and copy the returned `content` to its own clipboard.

## Clipboard status

`GET /api/v1/clipboard/status`

```json
{"module":"clipboard","enabled":true,"has_latest":true}
```

## Errors

Errors are JSON objects with an `error` string. `400` means malformed JSON, an invalid JSON
shape or empty content; `404` means no latest item; `413` means the configured byte limit
was exceeded; `503` means the system clipboard is unavailable or a Windows write failed;
`500` is an unexpected internal error. Unsupported methods return `405`.

## Compatibility and security

The `/api/v1` prefix is reserved for backward-compatible additions. New content types should
reuse the `Item` envelope and add a new MIME type. Breaking changes require `/api/v2` and an
ADR. The service currently has no authentication or encryption, so clients must use a trusted
LAN and must not send secrets.
