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

### 对端健康与剪贴板出站投递

当 `device.health_interval` 为正数（默认 `30s`）时，LocalBridge 会使用对端地址、端口和 peer Token，定期请求已配对
对端的 `GET /api/v1/system/capabilities`。设备列表会显示 `online` 或 `offline`；探测成功会在内存中更新 `last_seen`
和能力。

对于声明支持 `clipboard.text.push` 的已配对对端，本地 Windows 剪贴板事件会使用 peer Token POST 到对端的
`/api/v1/clipboard`。`source` 为 `remote` 的事件不会再次转发，因此两个 LocalBridge 主机之间不会形成回环。请求使用
上文定义的剪贴板 JSON 字段，远端响应的 `accepted` 只作为元数据记录到日志。

当前出站路径是尽力而为：有请求超时，但没有持久化队列、重试调度器、投递回执存储或离线重放。EventBus 缓冲区满时通知
可能被丢弃；这些保证属于 Phase 3 同步引擎。由于 iPhone 不运行常驻监听器，Phase 1 的 iPhone Shortcuts 仍然是主动 Pull。

当前任务查询接口是 `GET /api/v1/sync/jobs` 和 `GET /api/v1/sync/jobs/{id}`。Envelope 字段、状态转换、保留策略和限制请查看
[同步引擎基础](sync.zh-CN.md)。

## 文件分享与接收（Phase 4 基础）

文件模块将元数据保存到 `files.store_path`，将接收数据写入 `files.receive_dir`。启用认证时，管理接口需要配置的
Bearer Token 或 peer Token。即使 `security.auth_enabled` 为 `false`，所有 `/api/v1/files/...` 管理操作仍只允许回环
请求，局域网请求返回 `403`；公开能力 URL 仍可供手机访问。不能把关闭 Bearer Token 理解为允许远程管理文件。
`POST /api/v1/files/shares` 接收
`{"files":[{"path":"C:\\Users\\me\\file.txt"}]}`，一次多选创建一条分享记录。`GET /api/v1/files/shares`
列出记录但不返回源路径和 Token；`DELETE /api/v1/files/shares/{id}` 清除一条，`DELETE /api/v1/files/shares` 清除全部。
可选的 `Idempotency-Key` Header（或请求体 `idempotency_key`）会让分享创建重试返回原记录。创建响应包含 `id`、`token`、
`files`、`created_at`、`expires_at` 和 `url`；Token 是 256 位随机能力凭证。

`POST /api/v1/files/receivers` 创建短期上传链接。`GET /api/v1/files/receives` 列出已完成接收记录，
`DELETE /api/v1/files/receives/{id}` 同时删除记录和保存的文件。二维码/移动端分享 URL 是 `GET /share/{token}`，会
渲染响应式页面并为每个文件提供下载链接；`GET /share/{token}/metadata` 返回 JSON 元数据。文件通过
`GET /share/{token}/files/{file_id}` 下载并支持 HTTP `Range` 断点续传，响应包含 `Accept-Ranges: bytes` 和
`X-Content-SHA256`。

接收 URL 是 `GET /receive/{token}`。页面使用 `POST /receive/{token}/uploads` 发送
`{"name":"photo.jpg","size":123456,"sha256":"<可选的小写 sha256>"}`，再使用
`PUT /receive/{token}/uploads/{upload_id}` 和 `Content-Range: bytes start-end/total` 上传分片。单片最多 16 MiB，
且必须连续；对已保存范围重复发送相同字节是幂等的，跳过偏移或冲突重放返回 `409`。完成时校验声明的 SHA-256，
然后将生成的临时文件移动到 `files.receive_dir`。上传创建同样支持 `Idempotency-Key`。

`GET /receive/{token}/uploads/{upload_id}` 在校验接收 Token 后返回当前 `received_bytes`、`status` 和元数据。客户端在
超时或重启后可以使用它从已提交偏移继续。内置移动页面会根据接收 URL 和文件身份生成稳定幂等键，将上传 ID 保存在
`localStorage`，并从该状态接口恢复；不会为了计算客户端哈希而一次性将整个大文件加载到内存，完整文件的哈希由服务端
在上传完成时计算。

`GET /api/v1/files/shares/{id}/qr` 是与渲染器无关的二维码数据契约，不返回二维码图片：

```json
{"version":1,"type":"localbridge.share","url":"http://192.168.1.10:8899/share/<token>","expires_at":"..."}
```

未来 GUI 应在本地将 `url` 编码为二维码。本 Sprint 只提供契约，占位接口不代表离线二维码图片已经完成。

安全限制在存储前执行：源文件必须是普通且非符号链接文件；单文件、单次分享、接收配额和文件数限制由 `files.*` 控制；
上传名称不能包含路径分隔符、控制字符或 `..`；接收路径由随机 ID 生成，客户端不能提供。Token 会过期且不会写入日志。
公开的 `/share/` 和 `/receive/` 只有在 URL Token 作为能力凭证时才绕过管理认证；请将链接限制在可信局域网内。当前服务
仍然是纯 HTTP，不应暴露到公网。

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

device:
  health_interval: 30s
```

服务端使用常量时间比较 Token，并且不会把 Token 写入日志。只有启用认证后 peer Token 才会生效。这是 Phase 2 的过渡机制；
后续配对流程会自动配置和轮换管理凭据，不再要求用户直接编辑 YAML 中的密钥。

## 兼容性与安全

`/api/v1` 前缀用于向后兼容的增量扩展。新的内容类型应复用 `Item` 信封并增加 MIME 类型。破坏性变更需要使用
`/api/v2` 并编写 ADR。当前服务没有认证或加密，因此客户端必须使用可信局域网，不要发送密码等敏感信息。
