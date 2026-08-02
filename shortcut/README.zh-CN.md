# iPhone 快捷指令

[English version](README.md)

第一版有意采用两个简单的快捷指令。iPhone 是客户端，Windows 承载 HTTP API；两台设备必须位于同一个可信局域网。

## Push Clipboard（推送剪贴板）

创建快捷指令并添加以下操作：

1. **获取剪贴板**（Get Clipboard）。
2. **获取 URL 内容**（Get Contents of URL）。
3. URL：`http://WINDOWS_IP:8899/api/v1/clipboard`。
4. Method：`POST`。
5. 请求体类型：`JSON`。
6. JSON 字段：

```json
{
  "type": "text",
  "mime_type": "text/plain",
  "content": "Clipboard"
}
```

将 `Clipboard` 的值替换为“获取剪贴板”的输出。如有需要，可根据返回的 `accepted` 值添加通知。完整请求字段如下：

| 字段 | 必填 | 含义 |
|---|---:|---|
| `content` | 是* | 剪贴板文本，使用“获取剪贴板”的输出。 |
| `text` | 否 | `content` 的兼容性别名。 |
| `type` | 否 | Phase 1 使用 `text`。 |
| `mime_type` | 否 | Phase 1 使用 `text/plain`。 |
| `device_id` | 否 | 可选的来源设备标签，例如 `iphone-personal`。 |
| `id` | 否 | 可选的客户端项目 ID。 |
| `hash` | 否 | 可选的小写 SHA-256；缺失时由服务端计算。 |

\* 也可以使用非空的 `text` 别名代替 `content`。响应字段、大小限制和错误码请查看
[剪贴板模块与 API](../docs/clipboard.zh-CN.md)。

## Pull Clipboard（拉取剪贴板）

1. **获取 URL 内容**。
2. URL：`http://WINDOWS_IP:8899/api/v1/clipboard/latest`。
3. Method：`GET`。
4. 读取 JSON 的 `content` 字段。
5. **拷贝到剪贴板**（Copy to Clipboard）。

可以使用快捷指令自动化、Action Button、轻点背面或 Siri 触发。服务不会尝试让 iPhone 运行后台服务器，
因此符合 iOS 常规快捷指令执行模型。

## 连接检查

在同一局域网的浏览器中打开 `http://WINDOWS_IP:8899/api/v1/system/health`，响应应包含 `"status":"ok"`。
