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

响应还包含 `request_id`。每个响应都会在 `X-Request-ID` Header 中返回相同值。客户端可以发送自己的短
`X-Request-ID`；缺失或过长时由服务端生成。

## 能力发现

`GET /api/v1/system/capabilities`

客户端可以在发送内容前通过该接口选择兼容工作流：

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

能力名称是不可枚举的字符串。客户端必须容忍未知能力，并且在使用某项能力前不能假设它一定存在。

## 推送剪贴板

`POST /api/v1/clipboard`

当 `security.auth_enabled` 为 `true` 时，请发送 `Authorization: Bearer <token>`，可以使用管理 Token 或显式配对响应中
返回的 peer Token。健康检查无需认证，以便本地
操作人员诊断进程是否运行；剪贴板和能力接口需要认证。

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

## 设备与显式配对

`GET /api/v1/devices` 返回本地设备和已配对对端。响应不会包含对端 Token：

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

`POST /api/v1/devices/pair` 使用本地配置的 `security.pairing_code` 显式配对对端：

```json
{
  "code": "configured-local-pairing-code",
  "id": "iphone-personal",
  "name": "iPhone",
  "address": "192.168.1.20",
  "port": 8899,
  "capabilities": ["clipboard.text.push", "clipboard.text.pull"]
}
```

响应会返回对端元数据和生成的 peer Token。Token 只在配对响应中返回，不会出现在 list/get 响应或日志中。对同一个
设备 ID 再次配对会轮换 Token。`DELETE /api/v1/devices/{id}` 会撤销对端。本 Sprint 将注册表保存到
`device.registry_path`；静态加密存储以及自动 Token 配置/轮换属于后续 Phase 2 工作。

### 局域网发现

当 `discovery.enabled` 为 `true` 时，设备模块会在 `discovery.port`（默认 `8898`）上发送并监听有大小限制的 UDP/IPv4
广播。`GET /api/v1/devices/discovered` 返回最近的可达性提示。发现数据包包含协议版本、设备元数据、API 端口、能力和
nonce，不包含认证凭据。

发现到的设备不会加入 `peers`，不能访问受保护接口，也不会获得信任，直到显式配对成功。某些网络可能屏蔽组播/广播；
手动配对始终是回退方式。

## 错误

错误是包含 `error` 字符串的 JSON 对象。`400` 表示 JSON 格式错误、JSON 结构错误或内容为空；`404` 表示没有最新项目；
`413` 表示超过配置的字节限制；`503` 表示系统剪贴板不可用或 Windows 写入失败；`500` 表示未预期的内部错误。
`401` 表示缺少或无效的认证。不支持的方法返回 `405`。错误响应包含 `request_id`，并在 `X-Request-ID` 响应
Header 中返回同一个值。

## 安全配置

```yaml
security:
  auth_enabled: true
  bearer_token: "a-long-random-token-at-least-16-characters"
  pairing_code: "a-local-pairing-code"

discovery:
  enabled: false
  port: 8898
  announce_interval: 10s
```

服务端使用常量时间比较 Token，并且不会把 Token 写入日志。只有启用认证后 peer Token 才会生效。这是 Phase 2 的过渡机制；
后续配对流程会自动配置和轮换管理凭据，不再要求用户直接编辑 YAML 中的密钥。

## 兼容性与安全

`/api/v1` 前缀用于向后兼容的增量扩展。新的内容类型应复用 `Item` 信封并增加 MIME 类型。破坏性变更需要使用
`/api/v2` 并编写 ADR。当前服务没有认证或加密，因此客户端必须使用可信局域网，不要发送密码等敏感信息。
