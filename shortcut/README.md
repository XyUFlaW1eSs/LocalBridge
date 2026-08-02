# iPhone Shortcuts

[简体中文](README.zh-CN.md)

The first release intentionally uses two simple Shortcuts. The iPhone is a client; Windows
hosts the HTTP API. Both devices must be on the same trusted LAN.

## Push Clipboard

Create a Shortcut with these actions:

1. **Get Clipboard**.
2. **Get Contents of URL**.
3. URL: `http://WINDOWS_IP:8899/api/v1/clipboard`.
4. Method: `POST`.
5. Request body: `JSON`.
6. JSON fields:

```json
{
  "type": "text",
  "mime_type": "text/plain",
  "content": "Clipboard"
}
```

Replace the `Clipboard` value with the output of **Get Clipboard**. Add a notification for
the returned `accepted` value if desired. The complete request fields are:

| Field | Required | Meaning |
|---|---:|---|
| `content` | Yes* | Clipboard text; use the **Get Clipboard** output. |
| `text` | No | Compatibility alias for `content`. |
| `type` | No | Use `text` for Phase 1. |
| `mime_type` | No | Use `text/plain` for Phase 1. |
| `device_id` | No | Optional source-device label, such as `iphone-personal`. |
| `id` | No | Optional client item ID. |
| `hash` | No | Optional lowercase SHA-256; the server computes it when omitted. |

\* A non-empty `text` alias can be used instead of `content`. See
[Clipboard Module and API](../docs/clipboard.md) for response fields, limits and errors.

## Pull Clipboard

1. **Get Contents of URL**.
2. URL: `http://WINDOWS_IP:8899/api/v1/clipboard/latest`.
3. Method: `GET`.
4. Read the JSON `content` field.
5. **Copy to Clipboard**.

Use the phone's Shortcuts automation, Action Button, Back Tap or Siri to trigger these
actions. The service does not try to run an iPhone background server, which keeps the design
within iOS's normal Shortcut execution model.

## Connection check

Open `http://WINDOWS_IP:8899/api/v1/system/health` from a browser on the same LAN. The
response should contain `"status":"ok"`.
