# 协议与 API

[English version](protocol.md)

基础 URL：`http://<windows-ip>:8899`。

所有请求体都是 UTF-8 JSON，时间使用 RFC 3339 UTC。Phase 1 只支持文本。

## 健康检查

`GET /api/v1/system/health`

```json
{"status":"ok","service":"localbridge","time":"2026-08-02T00:00:00Z"}
```

## 推送剪贴板

`POST /api/v1/clipboard`

请求：

```json
{
  "type": "text",
  "mime_type": "text/plain",
  "content": "hello from iPhone",
  "device_id": "iphone-personal"
}
```

`text` 是兼容性别名，可替代 `content`。`hash` 和 `id` 可选；服务器会在缺失时计算 SHA-256 并创建 ID。

响应：

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

重复发送相同哈希时仍返回 HTTP 200，但 `accepted` 为 `false`。超过 `clipboard.max_text_bytes` 返回 `413`，
JSON 格式错误返回 `400`。

## 拉取剪贴板

`GET /api/v1/clipboard/latest`

返回最新项目。新进程在尚未收到剪贴板事件时返回 `404`。快捷指令应读取 `content` 字段并将其复制到 iPhone
剪贴板。

## Windows 本地剪贴板方向

当用户直接在 Windows 上复制文本时，监听器会读取文本并更新内存中的 `latest` 项目。LocalBridge 当前不知晓
iPhone 的对端地址，也不会自动向 iPhone 发 HTTP 请求。iPhone 必须运行 Pull 快捷指令主动 GET `latest`，再将
返回的 `content` 设置到自己的剪贴板。

## 剪贴板状态

`GET /api/v1/clipboard/status`

返回模块是否启用，以及内存中是否存在项目。

## 兼容性与未来版本

`/api/v1` 前缀用于向后兼容的增量扩展。新的内容类型应复用 `Item` 信封并增加 MIME 类型。破坏性变更需要使用
`/api/v2` 并编写 ADR。当前服务没有认证，因此不要在不可信局域网上发送敏感信息。
