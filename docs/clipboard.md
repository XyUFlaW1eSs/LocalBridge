# Clipboard Module and API

[简体中文](clipboard.zh-CN.md)

This document is the field-level reference for the Phase 1 clipboard module. The HTTP
wire contract is also summarized in [Protocol and API](protocol.md).

## Scope and data flow

Phase 1 supports UTF-8 text only. The Windows process keeps the latest item in memory and
bridges it in two directions:

```text
iPhone / LAN client --POST /clipboard--> LocalBridge --WriteText--> Windows clipboard
iPhone / LAN client <--GET /clipboard/latest-- LocalBridge <--Watch-- Windows clipboard
```

The second line is a pull flow. LocalBridge does not currently discover an iPhone, keep a
peer registry, or send an unsolicited HTTP request to another device.

## Endpoints

| Method | Path | Purpose | Request body |
|---|---|---|---|
| `POST` | `/api/v1/clipboard` | Accept remote clipboard text and write it to Windows | JSON object or raw text |
| `GET` | `/api/v1/clipboard/latest` | Return the latest in-memory item | None |
| `GET` | `/api/v1/clipboard/status` | Return module and availability status | None |

The default base URL is `http://<windows-ip>:8899`. All JSON is UTF-8. Timestamps are RFC
3339 UTC strings.

## Push request: `POST /api/v1/clipboard`

### JSON fields

The preferred request body is a JSON object. `content` is the only semantically required
value; all other fields are optional.

| Field | JSON type | Required | Default / fallback | Meaning |
|---|---|---:|---|---|
| `content` | string | Yes* | Falls back to `text` | UTF-8 clipboard text. An empty string is rejected. |
| `text` | string | No | Used only when `content` is missing or empty | Backward-compatible alias for `content`. Prefer `content`. |
| `type` | string | No | `text` | Logical content type. Phase 1 uses `text`; the server currently preserves a supplied value. |
| `mime_type` | string | No | `text/plain` | MIME type describing the payload. Phase 1 should use `text/plain`. |
| `device_id` | string | No | Server `device.id` (default `windows-pc`) | Identifier of the device that produced the request. It is metadata, not authentication. |
| `id` | string | No | `<device_id>-<unix-nanoseconds>` | Client item ID. The server generates one when omitted. |
| `hash` | string | No | Lowercase SHA-256 of `content` | Deduplication key. If supplied, the current implementation uses the supplied value as-is; clients should send the SHA-256 of the UTF-8 content. |

\* A request is valid when it contains a non-empty `content`, or a non-empty `text` alias.

Example with all metadata:

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

Minimal JSON:

```json
{
  "content": "hello from iPhone"
}
```

### Raw-text request

Clients may send plain UTF-8 text instead of JSON. Use `Content-Type: text/plain`; the
server treats the entire body as `content` and applies `type: text` and `mime_type:
text/plain`. A non-JSON body is also treated as raw text for compatibility.

```http
POST /api/v1/clipboard HTTP/1.1
Content-Type: text/plain

hello from iPhone
```

### Request limits and validation

- The maximum size is `clipboard.max_text_bytes`, 1 MiB (`1048576` bytes) by default.
- The limit is measured in UTF-8 bytes, not the number of Unicode characters.
- Empty content is rejected.
- A JSON request must be an object. Malformed JSON or an unsupported JSON shape is rejected.
- Phase 1 does not transport images, HTML, files, rich text or binary data.

## Push response

Successful requests return HTTP `200` with:

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

`accepted` is `true` when a new item becomes the latest item. If the request hash equals
the current latest hash, the server returns HTTP `200` with `accepted: false`; it does not
write the Windows clipboard again. The response `item` is the current item in both cases.

### Response `item` fields

| Field | JSON type | Meaning |
|---|---|---|
| `id` | string | Item identifier from the request or server-generated. |
| `type` | string | Logical content type; `text` in the Phase 1 flow. |
| `mime_type` | string | Payload MIME type; normally `text/plain`. |
| `content` | string | The clipboard text. |
| `hash` | string | Deduplication hash, normally lowercase SHA-256 of UTF-8 content. |
| `device_id` | string | Producing device metadata. |
| `source` | string | `remote` for an accepted HTTP push; `local` for a Windows watcher event. |
| `created_at` | string | UTC creation time in RFC 3339 format. |

## Pull response: `GET /api/v1/clipboard/latest`

The endpoint returns the same `item` shape directly, not wrapped in another object:

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

If the process has not observed any content yet, the endpoint returns HTTP `404`:

```json
{"error":"clipboard is empty"}
```

A phone Pull Shortcut should read `content` and copy that string to the iPhone clipboard.

## Status response: `GET /api/v1/clipboard/status`

```json
{
  "module": "clipboard",
  "enabled": true,
  "has_latest": true
}
```

- `module`: stable module name.
- `enabled`: whether the module is enabled by configuration.
- `has_latest`: whether an item is currently held in memory.

## Error responses

Errors use a JSON object with an `error` string.

| HTTP status | Typical condition | Example |
|---:|---|---|
| `400` | Malformed JSON, wrong JSON shape, or empty content | `{"error":"clipboard content is empty"}` |
| `404` | `latest` requested before any item exists | `{"error":"clipboard is empty"}` |
| `405` | Unsupported HTTP method | Standard `net/http` response |
| `413` | UTF-8 body exceeds `clipboard.max_text_bytes` | `{"error":"clipboard content exceeds configured limit"}` |
| `503` | OS clipboard is unsupported or Windows clipboard write failed | `{"error":"failed to write system clipboard: ..."}` |
| `500` | Unexpected internal failure | JSON error response |

## Lifecycle, deduplication and feedback-loop protection

1. A local Windows clipboard change is detected by the watcher, converted to an item with
   `source: local`, and stored as `latest`.
2. A remote POST is validated, deduplicated, written to the Windows clipboard, and stored
   with `source: remote`.
3. A successful remote write starts a two-second suppression window. This prevents the
   watcher from echoing the same remote write back into the module.
4. `latest` is memory-only and is lost when the process restarts.

## Logging and security

The service logs startup, watcher events, push/pull handling, duplicate detection, system
clipboard writes, HTTP status and errors to stdout by default. It logs metadata such as
hash, byte count and source, but does not log clipboard text content. Phase 1 has no
authentication or encryption; use only on a trusted LAN and do not send secrets.

## Platform implementation

`Platform` exposes `ReadText`, `WriteText` and `Watch`. Windows uses Win32 Unicode clipboard
APIs and `GetClipboardSequenceNumber`. Other operating systems use a safe no-op adapter so
the server remains buildable while native integrations are planned. The watcher is polling-
based and retries on its next interval when another Windows process temporarily owns the
clipboard.

## Configuration

```yaml
clipboard:
  enabled: true
  watch_interval: 300ms
  max_text_bytes: 1048576
```

`watch_interval` controls local polling frequency. `max_text_bytes` controls the maximum
request and locally observed text size.

## Known limitations

- No automatic Windows-to-iPhone push; the phone must pull.
- No images, HTML, files, rich text, encryption or device authentication.
- Latest data is not persisted across restarts.
- Clipboard access can transiently fail while another Windows process holds the clipboard.
