# 协议与 API

[English version](protocol.md)

完整的剪贴板字段参考请查看[剪贴板模块与 API](clipboard.zh-CN.md)。

基础 URL：`http://<windows-ip>:8899`。

Phase 1 只传输 UTF-8 文本。JSON 请求和响应使用 UTF-8，时间使用 RFC 3339 UTC 字符串。

## 健康检查

`GET /api/v1/system/health`

```json
{"status":"ok","service":"localbridge","time":"2026-08-02T00:00:00Z"}
```

## 推送剪贴板

`POST /api/v1/clipboard`

推荐请求体为 JSON 对象：

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

| 字段 | 类型 | 必填 | 默认值/含义 |
|---|---|---:|---|
| `content` | string | 是* | UTF-8 剪贴板文本；空内容会被拒绝。 |
| `text` | string | 否 | 兼容性别名；当 `content` 缺失或为空时使用。 |
| `type` | string | 否 | 默认 `text`；Phase 1 仅支持文本。 |
| `mime_type` | string | 否 | 默认 `text/plain`。 |
| `device_id` | string | 否 | 默认使用服务端 `device.id`；表示产生内容的设备。 |
| `id` | string | 否 | 缺失时使用设备 ID 与 Unix 纳秒时间戳生成。 |
| `hash` | string | 否 | 小写 SHA-256 去重键；缺失时由服务端计算。 |

\* 非空的 `text` 别名也可以满足内容要求。

使用 `Content-Type: text/plain` 时，也可以直接发送原始 UTF-8 文本；服务端会将其解释为 `content`，并使用
`type: text` 与 `mime_type: text/plain`。

默认最大大小是 `clipboard.max_text_bytes: 1048576` 字节，按 UTF-8 字节数计算。成功响应为 HTTP `200`：

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

`accepted: false` 表示 hash 与当前最新项目相同，因此不会再次写入 Windows 剪贴板。`item` 字段包括 `id`、`type`、
`mime_type`、`content`、`hash`、`device_id`、`source`（`remote` 或 `local`）和 `created_at`。

## 拉取剪贴板

`GET /api/v1/clipboard/latest`

直接返回最新的 `item`：

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

尚未有项目时返回 HTTP `404` 和 `{"error":"clipboard is empty"}`。

## Windows 本地剪贴板方向

当用户直接在 Windows 上复制文本时，监听器会读取文本并更新内存中的 `latest` 项目。LocalBridge 当前不知道
iPhone 的对端地址，也不会自动向手机发 HTTP 请求。iPhone 必须运行 Pull 快捷指令主动 GET `latest`，再将返回的
`content` 复制到自己的剪贴板。

## 剪贴板状态

`GET /api/v1/clipboard/status`

```json
{"module":"clipboard","enabled":true,"has_latest":true}
```

## 错误

错误是包含 `error` 字符串的 JSON 对象。`400` 表示 JSON 格式错误、JSON 结构错误或内容为空；`404` 表示没有最新项目；
`413` 表示超过配置的字节限制；`503` 表示系统剪贴板不可用或 Windows 写入失败；`500` 表示未预期的内部错误。
不支持的方法返回 `405`。

## 兼容性与安全

`/api/v1` 前缀用于向后兼容的增量扩展。新的内容类型应复用 `Item` 信封并增加 MIME 类型。破坏性变更需要使用
`/api/v2` 并编写 ADR。当前服务没有认证或加密，因此客户端必须使用可信局域网，不要发送密码等敏感信息。
