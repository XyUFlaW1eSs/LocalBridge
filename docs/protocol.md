# Protocol and API

[简体中文](protocol.zh-CN.md)

Base URL: `http://<windows-ip>:8899`.

All bodies are UTF-8 JSON. Times are RFC 3339 UTC. Phase 1 is text-only.

## Health

`GET /api/v1/system/health`

```json
{"status":"ok","service":"localbridge","time":"2026-08-02T00:00:00Z"}
```

## Push clipboard

`POST /api/v1/clipboard`

Request:

```json
{
  "type": "text",
  "mime_type": "text/plain",
  "content": "hello from iPhone",
  "device_id": "iphone-personal"
}
```

`text` is accepted as a compatibility alias for `content`. `hash` and `id` are optional;
the server computes a SHA-256 hash and creates an ID if omitted.

Response:

```json
{
  "accepted": true,
  "item": {
    "id": "iphone-personal-0000000000000",
    "type": "text",
    "mime_type": "text/plain",
    "content": "hello from iPhone",
    "hash": "<sha256>",
    "device_id": "iphone-personal",
    "source": "remote",
    "created_at": "2026-08-02T00:00:00Z"
  }
}
```

Sending the same hash again returns HTTP 200 with `accepted: false`. Content larger than
`clipboard.max_text_bytes` returns `413`; malformed JSON returns `400`.

## Pull clipboard

`GET /api/v1/clipboard/latest`

Returns the latest item. A fresh process without a clipboard event returns `404`. The Shortcut
should read the `content` field and copy it to the iPhone clipboard.

## Clipboard status

`GET /api/v1/clipboard/status`

Returns whether the module is enabled and whether an item is available in memory.

## Compatibility and future versions

The `/api/v1` prefix is reserved for backward-compatible additions. New content types should
reuse the `Item` envelope and add a new MIME type. Breaking changes require `/api/v2` and an
ADR. The service currently has no authentication, so clients must not send secrets over an
untrusted LAN.
