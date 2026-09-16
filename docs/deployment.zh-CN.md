# 部署手册（Windows）

[English version](deployment.md)

## 1. 构建

安装 Go 1.24 或更高版本，并在 Windows 上构建：

```powershell
go test ./...
go vet ./...
go build -trimpath -ldflags "-s -w" -o .\dist\localbridge.exe .\cmd\localbridge
```

## 2. 配置

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
notepad .\configs\config.yaml
.\dist\localbridge.exe -check-config -config .\configs\config.yaml
```

设置稳定的 `device.id`。局域网使用时保持 `server.host` 为 `0.0.0.0`，并选择未被占用的端口。如果配置以后
包含凭据，不要提交 `configs/config.yaml`。

当前根 Schema 为 `version: 1`。没有版本的旧文件视为版本 0，只在内存迁移，服务不会改写它。验证旧部署后请手动增加
`version: 1`。未来版本会被拒绝，而不会被猜测性解析。`-check-config` 会执行同样的严格加载，输出脱敏的生效 JSON 后退出，
不会启动 GUI 或 HTTP Server。

如需在不启动服务的情况下收集排障信息，可运行 `localbridge.exe -support-bundle .\\support.zip -config .\\configs\\config.yaml`。
ZIP 只包含脱敏配置、运行时信息和状态文件元数据，不包含凭据、设备身份、本地路径、TLS 证书/私钥路径或内容、载荷或状态正文；
目标文件已存在时不会覆盖。

在允许其他设备连接前，建议启用 Phase 2 的过渡认证：

```yaml
security:
  credential_protection: required
  credential_store_path: "data/credentials.json"
  auth_enabled: true
  bearer_token: ""
  bearer_token_ref: management
  pairing_code: ""
  pairing_code_ref: pairing
  peer_token_ttl: 720h
  token_overlap_ttl: 10m
```

请在仓库外生成 Token，例如在 PowerShell 中使用 `[guid]::NewGuid().ToString("N")`，再通过 stdin 写入（不使用管道时为隐藏终端输入）：

```powershell
"replace-with-a-long-random-token" | .\dist\localbridge.exe -config .\configs\config.yaml -credential-action set -credential-name management
"replace-with-a-local-pairing-code" | .\dist\localbridge.exe -config .\configs\config.yaml -credential-action set -credential-name pairing
.\dist\localbridge.exe -config .\configs\config.yaml -credential-action status -credential-name management
```

管理 Token 至少 16 个字符。set/status/delete 都不接受秘密参数，也不会输出秘密值。凭据存储由当前用户 DPAPI 保护，
具有版本/大小限制，以仅所有者权限原子替换；更新一个条目不会破坏其他条目。删除使用 `-credential-action delete`。

如需配对另一台 LocalBridge 主机，设置本地 `security.pairing_code`，然后向 `POST /api/v1/devices/pair` 提交对端 ID、
地址、端口和能力，并在对端安全保存返回的 peer Token。它会在 `peer_token_ttl` 后过期，应在到期前通过
`POST /api/v1/devices/{id}/token/rotate` 轮换；旧凭据只在短暂重叠期继续有效。在 Windows `auto` 或 `required` 下，
注册表 v4 只持久化当前用户 DPAPI 密文形式的当前/上一代 peer Token。v3 明文注册表只有在全部 Token 保护成功后才原子改写；
保护或解密失败会中止启动并保留原文件。版本 1 注册表中缺失持久化 Token 的记录会变为
`repair_required`，需要重新配对或由本机轮换。只有确认允许在配置端口上使用 UDP 广播时才启用 `discovery.enabled`；
发现不会授予信任。正数的 `device.health_interval` 会启用尽力而为的对端健康检查和本地剪贴板出站。

`credential_protection: auto` 与 `required` 需要 Windows DPAPI。其他操作系统会明确报告生产保护不受支持并拒绝启动；
只有显式 `disabled` 才允许旧式明文持久化。`disabled` 不是加密，base64 编码也绝不称为保护。禁止把受保护的 v4 注册表
降级为 disabled。Windows `auto` 仍兼容旧 YAML 内联值，但不会自动迁移或删除它们；改为引用后必须由用户手动清理。

### TLS 传输

TLS 默认关闭。启用时必须配置匹配的证书/私钥：

```yaml
server:
  tls_enabled: true
  tls_cert_file: "C:\\path\\to\\localbridge.crt"
  tls_key_file: "C:\\path\\to\\localbridge.key"
```

应用会在模块组合前验证两个文件，并只启动 HTTPS，最低 TLS 版本为 1.2（优先 1.3）；不会静默回退 HTTP。
叶证书的小写 SHA-256 会出现在 capabilities 和 discovery 中；`secure: true` 的配对记录必须保存该指纹。
Discovery 只是提示，不会建立信任。对端传输可以通过精确指纹固定自签名证书，但 Windows 浏览器/WebView2 和 iPhone
Safari/Shortcuts 可能拒绝未安装信任的证书，请将私有 CA 或证书安装到平台信任库。首次指纹确认/分发 UX、自动证书轮换和
iPhone 信任安装流程仍是后续工作。

## 3. 防火墙

仅在 Private 配置文件中允许 TCP 8899 入站；如果修改端口，请同步替换命令中的端口：

```powershell
New-NetFirewallRule -DisplayName "LocalBridge (Private LAN)" `
  -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8899 -Profile Private
```

不要为 Public 配置文件创建规则，也不要从路由器将此服务端口转发到公网。

## 4. 运行与验证

```powershell
.\dist\localbridge.exe -config .\configs\config.yaml
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/health
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/capabilities `
  -Headers @{ Authorization = "Bearer replace-with-a-long-random-token" }
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/config
```

健康检查有意保持免认证，以便本地诊断。启用认证后，剪贴板和能力请求（包括 iPhone 快捷指令）都必须添加同一个
`Authorization: Bearer <token>` Header。每个响应都会包含 `X-Request-ID`；报告故障时请保留该值。

生效配置响应已经脱敏：凭据只显示是否已配置，不返回配置文件路径。关闭认证时本机回环可直接读取、远程访问被拒绝；
启用认证后所有请求都必须使用管理 Token，peer Token 会被拒绝。

`/api/v1/files/` 下的文件管理接口还有额外边界：当 `security.auth_enabled: false` 时，只接受回环请求，来自局域网
的管理请求返回 `403`；启用认证后，管理操作必须使用 Bearer Token 或已配对的 peer Token。`/share/<token>` 和
`/receive/<token>` 页面是供手机访问的公开能力 URL，请只在可信局域网内使用随机且会过期的链接。二维码接口同时提供
渲染器无关的 JSON URL（`/api/v1/files/shares/<id>/qr`）和本地生成的 PNG 图片
（`/api/v1/files/shares/<id>/qr.png`）。TLS 关闭时内嵌浏览器 GUI 位于 `http://127.0.0.1:8899/app/`，启用时位于
`https://127.0.0.1:8899/app/`，包含分享、接收记录和设置三个区域。
浏览器上传使用 `POST /api/v1/files/browser-shares`；一个 multipart 批次创建一条分享，文件写入配置的 `files.share_dir`
（默认 `data/shared`），不会使用客户端提供的本地路径。启用认证后，本机回环管理请求仍供本地 GUI 使用；局域网管理客户端仍
必须提供 Bearer Token 或已配对的 peer Token。

GUI 设置保存于 `settings.store_path`（默认 `data/settings.json`），是版本化、非敏感、以严格权限原子写入的文档。在 Windows 上，
`auto_start` 与 `explorer_context_menu` 现会同步当前用户 HKCU 注册表；原生托盘提供分享、接收记录、设置与退出操作。文件完成事件
可以产生托盘通知和声音。`auto_accept` 已生效：关闭时，新上传在“接受记录”等待允许或拒绝，等待期间不会接收文件字节。
Windows WebView2 宿主会执行 `minimize_to_tray`；运行时不可用时回退到系统浏览器，此时无法拦截浏览器关闭。二维码图片由 MIT
许可的 `github.com/skip2/go-qrcode` 在本地生成，部署不需要 CDN 或公网二维码服务。

需要持续查看日志时，请在 PowerShell 中运行：

```powershell
.\scripts\run.ps1 -Executable .\dist\localbridge-v0.1.0\localbridge.exe -Config .\configs\config.yaml
```

程序会在当前控制台输出模块启动、HTTP 请求、请求 ID、能力、剪贴板 Push/Pull、去重和 Win32 写入错误；不会输出
剪贴板正文或认证 Token。

在 iPhone 上将快捷指令 URL 设置为 Windows 私有 IPv4 地址。依次测试 Push、Pull，然后在 Windows 上直接复制
文本，并在等待监听间隔后确认 latest 接口已更新。

## 5. Windows 外壳集成

在 `/app/#settings` 中启用“开机自启”或“Explorer 右键菜单”会创建当前用户 HKCU 项，不需要管理员权限。Explorer 使用
`localbridge.exe -share <files>` 调用程序，并尽可能复用正在运行的回环服务。托盘菜单可打开各 GUI 区域或请求优雅退出。
程序必须运行在交互式用户会话，因为剪贴板和托盘属于该桌面。这不是 Windows Service。GUI 使用系统安装的 Microsoft Edge WebView2
Runtime 和 MIT 许可的纯 Go `github.com/jchv/go-webview2` 宿主；按设置关闭原生窗口会隐藏到托盘，可从托盘恢复或正常退出。

## 回滚

停止进程，替换为上一版本的可执行文件，并使用相同配置重启。Phase 1 的 latest 项目只保存在内存中，因此回滚
不需要迁移存储。
