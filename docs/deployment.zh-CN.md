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
```

设置稳定的 `device.id`。局域网使用时保持 `server.host` 为 `0.0.0.0`，并选择未被占用的端口。如果配置以后
包含凭据，不要提交 `configs/config.yaml`。

在允许其他设备连接前，建议启用 Phase 2 的过渡认证：

```yaml
security:
  auth_enabled: true
  bearer_token: "replace-with-a-long-random-token"
```

请在仓库外生成 Token，例如在 PowerShell 中使用 `[guid]::NewGuid().ToString("N")`。Token 至少需要 16 个字符，
实际使用时应采用更长的随机值。服务端不会记录 Token。

如需配对另一台 LocalBridge 主机，设置本地 `security.pairing_code`，然后向 `POST /api/v1/devices/pair` 提交对端 ID、
地址、端口和能力，并在对端安全保存返回的 peer Token。只有确认允许在配置端口上使用 UDP 广播时才启用
`discovery.enabled`；发现不会授予信任。正数的 `device.health_interval` 会启用尽力而为的对端健康检查和本地剪贴板出站。

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
```

健康检查有意保持免认证，以便本地诊断。启用认证后，剪贴板和能力请求（包括 iPhone 快捷指令）都必须添加同一个
`Authorization: Bearer <token>` Header。每个响应都会包含 `X-Request-ID`；报告故障时请保留该值。

`/api/v1/files/` 下的文件管理接口还有额外边界：当 `security.auth_enabled: false` 时，只接受回环请求，来自局域网
的管理请求返回 `403`；启用认证后，管理操作必须使用 Bearer Token 或已配对的 peer Token。`/share/<token>` 和
`/receive/<token>` 页面是供手机访问的公开能力 URL，请只在可信局域网内使用随机且会过期的链接。二维码接口同时提供
渲染器无关的 JSON URL（`/api/v1/files/shares/<id>/qr`）和本地生成的 PNG 图片
（`/api/v1/files/shares/<id>/qr.png`）。内嵌浏览器 GUI 位于 `http://127.0.0.1:8899/app/`，包含分享、接收记录和设置三个区域。
浏览器上传使用 `POST /api/v1/files/browser-shares`；一个 multipart 批次创建一条分享，文件写入配置的 `files.share_dir`
（默认 `data/shared`），不会使用客户端提供的本地路径。启用认证后，本机回环管理请求仍供本地 GUI 使用；局域网管理客户端仍
必须提供 Bearer Token 或已配对的 peer Token。

GUI 设置保存于 `settings.store_path`（默认 `data/settings.json`），是版本化、非敏感、以严格权限原子写入的文档。当前设置为未来的
开机启动、托盘、Explorer 右键菜单、自动接收和声音行为提供契约；Windows 原生效果明确推迟到 Sprint 4.2B。二维码图片由 MIT
许可的 `github.com/skip2/go-qrcode` 在本地生成，部署不需要 CDN 或公网二维码服务。

需要持续查看日志时，请在 PowerShell 中运行：

```powershell
.\scripts\run.ps1 -Executable .\dist\localbridge-v0.1.0\localbridge.exe -Config .\configs\config.yaml
```

程序会在当前控制台输出模块启动、HTTP 请求、请求 ID、能力、剪贴板 Push/Pull、去重和 Win32 写入错误；不会输出
剪贴板正文或认证 Token。

在 iPhone 上将快捷指令 URL 设置为 Windows 私有 IPv4 地址。依次测试 Push、Pull，然后在 Windows 上直接复制
文本，并在等待监听间隔后确认 latest 接口已更新。

## 5. 服务安装

Windows 原生服务、开机启动和托盘集成明确不属于 Sprint 4.2A。首次部署时应在受监管的用户会话中运行程序，因为 Windows
剪贴板访问属于交互式桌面会话。Sprint 4.2B 在产生系统效果前必须保留这一会话要求，并记录安全边界。

## 回滚

停止进程，替换为上一版本的可执行文件，并使用相同配置重启。Phase 1 的 latest 项目只保存在内存中，因此回滚
不需要迁移存储。
