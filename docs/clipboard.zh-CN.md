# 剪贴板模块与 API

[English version](clipboard.md)

本文是 Phase 1 剪贴板模块的字段级参考。HTTP 线协议也在[协议与 API](protocol.zh-CN.md)中有摘要说明。

## 范围与数据流

Phase 1 只支持 UTF-8 文本。Windows 进程在内存中保存最新项目，并通过以下两条方向完成桥接：

```text
iPhone / 局域网客户端 --POST /clipboard--> LocalBridge --WriteText--> Windows 剪贴板
iPhone / 局域网客户端 <--GET /clipboard/latest-- LocalBridge <--Watch-- Windows 剪贴板
```

第二条是“主动拉取”流程。LocalBridge 当前不会发现 iPhone、维护对端注册表，也不会主动向其他设备发起 HTTP 请求。

## 接口一览

| 方法 | 路径 | 用途 | 请求体 |
|---|---|---|---|
| `POST` | `/api/v1/clipboard` | 接收远程剪贴板文本并写入 Windows | JSON 对象或原始文本 |
| `GET` | `/api/v1/clipboard/latest` | 返回内存中的最新项目 | 无 |
| `GET` | `/api/v1/clipboard/status` | 返回模块及可用状态 | 无 |

默认基础 URL 为 `http://<windows-ip>:8899`。所有 JSON 使用 UTF-8，时间使用 RFC 3339 UTC 字符串。

## 推送请求：`POST /api/v1/clipboard`

### JSON 字段

推荐请求体为 JSON 对象。只有 `content` 在语义上是必需的，其他字段均为可选。

| 字段 | JSON 类型 | 必填 | 默认值/回退 | 含义 |
|---|---|---:|---|---|
| `content` | string | 是* | 回退到 `text` | UTF-8 剪贴板文本。空字符串会被拒绝。 |
| `text` | string | 否 | 仅在 `content` 缺失或为空时使用 | `content` 的兼容性别名。推荐使用 `content`。 |
| `type` | string | 否 | `text` | 逻辑内容类型。Phase 1 使用 `text`；当前服务会保留客户端传入值。 |
| `mime_type` | string | 否 | `text/plain` | 描述载荷的 MIME 类型。Phase 1 应使用 `text/plain`。 |
| `device_id` | string | 否 | 服务端 `device.id`（默认 `windows-pc`） | 产生该请求的设备标识。它只是元数据，不是认证信息。 |
| `id` | string | 否 | `<device_id>-<Unix纳秒时间戳>` | 客户端项目 ID。缺失时由服务端生成。 |
| `hash` | string | 否 | `content` 的小写 SHA-256 | 去重键。当前实现会原样使用客户端提供的值；客户端应发送 UTF-8 内容对应的 SHA-256。 |

\* 请求包含非空 `content`，或包含非空 `text` 别名时才有效。

包含完整元数据的示例：

```json
{
  "id": "iphone-personal-1710000000000000000",
  "type": "text",
  "mime_type": "text/plain",
  "content": "hello from iPhone",
  "hash": "<content的小写sha256>",
  "device_id": "iphone-personal"
}
```

最小 JSON：

```json
{
  "content": "hello from iPhone"
}
```

### 原始文本请求

客户端也可以不发送 JSON，直接发送 UTF-8 文本。请使用 `Content-Type: text/plain`；服务端会把整个请求体
作为 `content`，并自动使用 `type: text` 与 `mime_type: text/plain`。为了兼容，非 JSON 请求体也会按原始文本处理。

```http
POST /api/v1/clipboard HTTP/1.1
Content-Type: text/plain

hello from iPhone
```

### 请求限制与校验

- 最大大小由 `clipboard.max_text_bytes` 控制，默认是 1 MiB（`1048576` 字节）。
- 限制按 UTF-8 字节数计算，不是 Unicode 字符数。
- 空内容会被拒绝。
- JSON 请求必须是对象；格式错误或不支持的 JSON 结构会被拒绝。
- Phase 1 不传输图片、HTML、文件、富文本或二进制数据。

## 推送响应

成功请求返回 HTTP `200`：

```json
{
  "accepted": true,
  "item": {
    "id": "iphone-personal-1710000000000000000",
    "type": "text",
    "mime_type": "text/plain",
    "content": "hello from iPhone",
    "hash": "<小写sha256>",
    "device_id": "iphone-personal",
    "source": "remote",
    "created_at": "2026-08-02T00:00:00Z"
  }
}
```

当新项目成为最新项目时，`accepted` 为 `true`。如果请求的 hash 与当前最新项目相同，服务端仍返回 HTTP `200`，
但 `accepted: false`，且不会再次写入 Windows 剪贴板。两种情况下响应中的 `item` 都是当前项目。

### 响应 `item` 字段

| 字段 | JSON 类型 | 含义 |
|---|---|---|
| `id` | string | 请求中的项目 ID，或服务端生成的 ID。 |
| `type` | string | 逻辑内容类型；Phase 1 流程为 `text`。 |
| `mime_type` | string | 载荷 MIME 类型；通常为 `text/plain`。 |
| `content` | string | 剪贴板文本。 |
| `hash` | string | 去重 hash；通常是 UTF-8 内容的小写 SHA-256。 |
| `device_id` | string | 产生该项目的设备元数据。 |
| `source` | string | HTTP 推送被接受时为 `remote`；Windows 监听器发现时为 `local`。 |
| `created_at` | string | RFC 3339 格式的 UTC 创建时间。 |

## 拉取响应：`GET /api/v1/clipboard/latest`

该接口直接返回上述 `item` 结构，不再额外包装一层：

```json
{
  "id": "windows-pc-1710000000000000000",
  "type": "text",
  "mime_type": "text/plain",
  "content": "copied on Windows",
  "hash": "<小写sha256>",
  "device_id": "windows-pc",
  "source": "local",
  "created_at": "2026-08-02T00:00:00Z"
}
```

如果进程尚未观察到任何内容，接口返回 HTTP `404`：

```json
{"error":"clipboard is empty"}
```

手机端 Pull 快捷指令应读取 `content` 字段，并将该字符串复制到 iPhone 剪贴板。

## 状态响应：`GET /api/v1/clipboard/status`

```json
{
  "module": "clipboard",
  "enabled": true,
  "has_latest": true
}
```

- `module`：稳定的模块名称。
- `enabled`：模块是否由配置启用。
- `has_latest`：内存中当前是否有项目。

## 错误响应

错误使用包含 `error` 字符串的 JSON 对象。

| HTTP 状态码 | 常见原因 | 示例 |
|---:|---|---|
| `400` | JSON 格式错误、JSON 结构错误或内容为空 | `{"error":"clipboard content is empty"}` |
| `404` | 尚未有项目时请求 `latest` | `{"error":"clipboard is empty"}` |
| `405` | 不支持的 HTTP 方法 | `net/http` 标准响应 |
| `413` | UTF-8 请求体超过 `clipboard.max_text_bytes` | `{"error":"clipboard content exceeds configured limit"}` |
| `503` | 操作系统不支持剪贴板，或 Windows 写入失败 | `{"error":"failed to write system clipboard: ..."}` |
| `500` | 其他未预期的内部错误 | JSON 错误响应 |

## 生命周期、去重与反馈环保护

1. Windows 本地剪贴板发生变化时，监听器读取文本，生成 `source: local` 的项目并保存为 `latest`。
2. 远程 POST 经过校验和去重后写入 Windows 剪贴板，并以 `source: remote` 保存。
3. 远程写入成功后启动两秒抑制窗口，防止监听器把同一次远程写入再次回传到模块。
4. `latest` 只保存在内存中，进程重启后丢失。

## 日志与安全

服务默认向 stdout 输出启动、监听器、推送/拉取、重复检测、系统剪贴板写入、HTTP 状态和错误日志。日志会记录
hash、字节数、来源等元数据，但不会记录剪贴板正文。Phase 1 没有认证和加密，只应在可信局域网中使用，不要发送密码等敏感信息。

## 平台实现

`Platform` 提供 `ReadText`、`WriteText` 和 `Watch`。Windows 使用 Win32 Unicode 剪贴板 API 与
`GetClipboardSequenceNumber`。其他操作系统使用安全的空适配器，以便服务器仍可构建，等待未来的原生集成。
监听器采用轮询；当其他 Windows 进程暂时占用剪贴板时，会在下一次间隔重试。

## 配置

```yaml
clipboard:
  enabled: true
  watch_interval: 300ms
  max_text_bytes: 1048576
```

`watch_interval` 控制本地轮询频率，`max_text_bytes` 控制请求文本和本地观察文本的最大字节数。

## 已知限制

- 没有 Windows 到 iPhone 的自动推送；手机必须主动拉取。
- 不支持图片、HTML、文件、富文本、加密或设备认证。
- 最新数据不会跨进程重启持久化。
- 当其他 Windows 进程占用剪贴板时，访问可能暂时失败。
