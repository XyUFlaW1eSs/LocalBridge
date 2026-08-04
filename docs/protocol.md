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

The response also includes `request_id`. Every response carries the same value in the
`X-Request-ID` header. Clients may send their own short `X-Request-ID`; the server generates
one when it is missing or too long.

## Capabilities

`GET /api/v1/system/capabilities`

This endpoint lets a client choose a compatible workflow before sending content:

```json
{
  "service": "localbridge",
  "version": "v0.1.0",
  "api_version": "v1",
  "protocol_version": 1,
  "request_id": "<request-id>",
  "device": {"id": "windows-pc", "name": "LocalBridge Windows"},
  "capabilities": ["system.health", "system.capabilities", "clipboard.text.push", "clipboard.text.pull"]
}
```

Capability names are opaque strings. Clients must tolerate unknown capabilities and must not
assume that a capability exists without checking this endpoint.

## Push clipboard

`POST /api/v1/clipboard`

When `security.auth_enabled` is true, send `Authorization: Bearer <token>`. The configured
management token or a peer token returned from explicit pairing is accepted. Health remains
available without authentication so a local operator can diagnose whether the process is up;
clipboard and capabilities requests require authentication.

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

## Devices and explicit pairing

`GET /api/v1/devices` returns the local device and paired peers. Peer tokens are never included:

```json
{
  "local": {"id": "windows-pc", "name": "LocalBridge Windows"},
  "peers": [
    {
      "id": "iphone-personal",
      "name": "iPhone",
      "address": "192.168.1.20",
      "port": 8899,
      "capabilities": ["clipboard.text.push"],
      "status": "paired",
      "paired_at": "2026-08-04T00:00:00Z",
      "last_seen": "2026-08-04T00:00:00Z"
    }
  ]
}
```

`POST /api/v1/devices/pair` explicitly pairs a peer using the locally configured
`security.pairing_code`:

```json
{
  "code": "one-time-or-local-pairing-code",
  "id": "iphone-personal",
  "name": "iPhone",
  "address": "192.168.1.20",
  "port": 8899,
  "capabilities": ["clipboard.text.push", "clipboard.text.pull"]
}
```

The response returns the peer metadata and a generated peer token. The token is returned only
by the pairing response and is not returned by list/get endpoints or written to logs. Pairing
the same device ID rotates its token. `DELETE /api/v1/devices/{id}` revokes a peer. This Sprint
stores the registry at `device.registry_path`; encrypted-at-rest storage and automatic token
provisioning/rotation are later Phase 2 work.

### LAN discovery

When `discovery.enabled` is true, the device module sends and listens for bounded UDP/IPv4
announcements on `discovery.port` (default `8898`). `GET /api/v1/devices/discovered` returns
recent reachability hints. Discovery packets contain a protocol version, device metadata,
API port, capabilities and a nonce; they do not contain authentication credentials.

Discovered devices are not added to `peers`, cannot access protected APIs and are not trusted
until the explicit pairing flow succeeds. Multicast/broadcast may be blocked by some networks;
manual pairing remains the fallback.

### Peer health and outbound clipboard delivery

When `device.health_interval` is positive (default `30s`), LocalBridge periodically requests
`GET /api/v1/system/capabilities` from paired peers that have an address, port and peer token.
The peer status is reported as `online` or `offline` in the device list; a successful probe
updates `last_seen` and capabilities in memory.

For a paired peer advertising `clipboard.text.push`, a local Windows clipboard event is sent
to `POST /api/v1/clipboard` using the peer token. Events whose source is `remote` are not
forwarded again, which prevents a two-host echo loop. The request uses the same clipboard JSON
fields documented above and the remote response's `accepted` value is logged as metadata.

This first outbound path is best-effort: it has a bounded request timeout but no durable queue,
retry scheduler, delivery receipt store or offline replay. EventBus notifications can be dropped
when a subscriber is full. Those guarantees belong to the Phase 3 sync engine. iPhone Shortcuts
remain an explicit Pull workflow because iPhone does not run a persistent listener in Phase 1.

The current job inspection endpoints are `GET /api/v1/sync/jobs` and
`GET /api/v1/sync/jobs/{id}`. See [Sync Engine Foundation](sync.md) for the Envelope fields,
state transitions, retention and limitations.

## Errors

Errors are JSON objects with an `error` string. `400` means malformed JSON, an invalid JSON
shape or empty content; `404` means no latest item; `413` means the configured byte limit
was exceeded; `503` means the system clipboard is unavailable or a Windows write failed;
`500` is an unexpected internal error. `401` means authentication is missing or invalid.
Unsupported methods return `405`. Error responses include `request_id` and the same value in
the `X-Request-ID` response header.

## Security configuration

```yaml
security:
  auth_enabled: true
  bearer_token: "a-long-random-token-at-least-16-characters"
  pairing_code: "a-local-pairing-code"

discovery:
  enabled: false
  port: 8898
  announce_interval: 10s

device:
  health_interval: 30s
```

The token is compared in constant time and is never written to logs. Peer tokens are accepted
only after authentication is enabled. This configuration is a Phase 2 transition mechanism;
later pairing work will provision and rotate the management credential without requiring users
to edit a secret directly in YAML.

## Compatibility and security

The `/api/v1` prefix is reserved for backward-compatible additions. New content types should
reuse the `Item` envelope and add a new MIME type. Breaking changes require `/api/v2` and an
ADR. The service currently has no authentication or encryption, so clients must use a trusted
LAN and must not send secrets.
