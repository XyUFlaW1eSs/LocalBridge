# LocalBridge

语言： [English](README.md) · [简体中文](README.zh-CN.md)

LocalBridge 是一个轻量、以本地网络为优先的局域网协作平台。当前第一个可用模块通过两个 iPhone
快捷指令，在 Windows 电脑与 iPhone 之间同步文本剪贴板内容。不需要云账号或中继服务。

> 状态：Phase 0、Phase 1 / Sprint 1、Phase 2 基础、Phase 3.1、Phase 4.1、内嵌文件 GUI 与
> Sprint 4.2B Windows 外壳基础已交付。URL/图片模块、原生宿主窗口关闭最小化语义和其余 v1.0 路线仍在开发中。

## 功能概览

```text
iPhone 快捷指令 --HTTP POST--> Windows 上的 LocalBridge --Win32--> Windows 剪贴板
iPhone 快捷指令 <--HTTP GET--- Windows 上的 LocalBridge <--Win32-- Windows 剪贴板
```

- `POST /api/v1/clipboard`：接收来自 iPhone 快捷指令的文本。
- `GET /api/v1/clipboard/latest`：返回 Pull 快捷指令所需的最新内容。
- Windows 监听器轮询 Win32 剪贴板序列号，并发布本地变化。
- 使用 SHA-256 内容哈希避免重复更新和剪贴板反馈循环。
- 通过 EventBus 与模块边界，将文件、图片和通知功能保持为独立模块。

Windows 本地剪贴板变化目前不会主动向 iPhone 发 HTTP 请求。它会更新内存中的 `latest`，然后由 iPhone Pull
快捷指令主动 GET；这是 iOS 后台限制下的 Phase 1 设计。已配对的其他 LocalBridge 桌面主机现在可以通过尽力而为的
出站路径接收本地剪贴板事件；持久化队列和重试语义计划在 Phase 3 实现。

本地 Web GUI 位于 `http://127.0.0.1:8899/app/`，包含文件分享、接受记录和持久化设置三个区域。浏览器通过 multipart
上传到服务端受控的 `files.share_dir`，不会发送或显示真实本地路径。分享页支持拖拽、多选、可展开文件行、复制 URL 和
离线生成 PNG 二维码。Windows 现已支持当前用户级开机启动与 Explorer 右键菜单同步、原生托盘、多文件 `-share` 入口和完成通知。
GUI 仍由浏览器承载，因此 `minimize_to_tray` 设置虽然会持久化，但暂时不能拦截浏览器窗口的关闭按钮。

运行日志会输出到当前控制台。发布包可使用 `scripts/run.ps1` 前台运行，这样能看到每次 Push、Pull、去重和
剪贴板写入的诊断信息；日志不会记录剪贴板正文。

## Windows 快速开始

要求：Go 1.24 或更高版本，以及可信的私有局域网。当前服务没有认证或 TLS，不能绑定到不可信网络。

```powershell
Copy-Item configs/config.example.yaml configs/config.yaml
go run ./cmd/localbridge -config configs/config.yaml
```

默认监听地址为 `0.0.0.0:8899`。在 Windows 电脑上验证：

```powershell
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/health
```

然后将 [`shortcut/README.md`](shortcut/README.md) 中的 `WINDOWS_IP` 替换为 Windows 私有 IPv4 地址，
并创建 Push Clipboard 与 Pull Clipboard 快捷指令。

## 开发命令

```powershell
go test ./...
go vet ./...
gofmt -w cmd internal
go build ./cmd/localbridge
```

本地开发可使用 `configs/config.dev.yaml`。非 Windows 构建仍会启动 HTTP 服务，但使用空的剪贴板适配器，
便于在 CI 和其他平台测试核心包。

## 仓库结构

| 路径 | 职责 |
| --- | --- |
| `cmd/localbridge` | 进程入口、命令行参数和信号生命周期 |
| `internal/app` | 运行时组合与关闭流程 |
| `internal/config` | YAML 配置与校验 |
| `internal/eventbus` | 类型化事件名与非阻塞订阅 |
| `internal/module` | 稳定的模块生命周期与路由注册 |
| `internal/modules/device` | 设备注册表、显式配对和对端元数据 |
| `internal/transport` | 有边界的认证 HTTP JSON 对端传输 |
| `internal/syncstore` | 带版本的 Envelope 与持久化同步任务状态 |
| `internal/server` | 标准库 HTTP 服务与健康接口 |
| `internal/modules/clipboard` | 剪贴板 API、去重和平台适配器 |
| `internal/modules/files` | 分享/接收存储、Range 传输、浏览器上传和二维码 PNG |
| `internal/modules/settings` | 版本化非敏感 GUI 设置存储和 API |
| `internal/native` | Windows 开机启动、Explorer 命令、托盘和完成通知适配器 |
| `internal/web` | 内嵌 vanilla HTML/CSS/JS GUI，不依赖 CDN |
| `docs` | 架构、协议、部署与开发规范 |
| `shortcut` | iPhone 快捷指令配置与请求示例 |

## 安全边界

文件传输基础使用有过期时间的 capability URL 提供移动端公开页面。文件管理 API 只允许回环请求，或允许已经通过全局/已配对
对端认证边界的请求。服务仍只适用于可信的家庭/办公局域网，并应将 Windows 防火墙限制在 Private 网络配置文件；不要将 8899
端口或分享 URL 暴露到公网。

## 文档

- [架构](docs/architecture.zh-CN.md)
- [协议与 API](docs/protocol.zh-CN.md)
- [同步引擎基础](docs/sync.zh-CN.md)
- [剪贴板模块设计](docs/clipboard.zh-CN.md)
- [部署手册](docs/deployment.zh-CN.md)
- [开发指南](docs/developer-guide.zh-CN.md)
- [编码风格](docs/coding-style.zh-CN.md)
- [Sprint 1 交付](docs/sprints/sprint-1.zh-CN.md)
- [Sprint 2.1 交付](docs/sprints/sprint-2.1.zh-CN.md)
- [Sprint 2.2 交付](docs/sprints/sprint-2.2.zh-CN.md)
- [Sprint 2.3 交付](docs/sprints/sprint-2.3.zh-CN.md)
- [Sprint 2.4 交付](docs/sprints/sprint-2.4.zh-CN.md)
- [Sprint 2.5 交付](docs/sprints/sprint-2.5.zh-CN.md)
- [Sprint 3.1 交付](docs/sprints/sprint-3.1.zh-CN.md)
- [Sprint 4.1 文件传输](docs/sprints/sprint-4.1-file-transfer.zh-CN.md)
- [Sprint 4.2 桌面 GUI](docs/sprints/sprint-4.2-desktop-gui.zh-CN.md)
- [Sprint 4.2A 交付](docs/sprints/sprint-4.2a.zh-CN.md)
- [Sprint 4.2B Windows 外壳基础](docs/sprints/sprint-4.2b.zh-CN.md)
- [产品规划与能力地图](docs/product-plan.zh-CN.md)
- [文件分享产品计划](docs/file-sharing-product-plan.zh-CN.md)
- [工程流程与子任务职责](docs/engineering-workflow.zh-CN.md)
- [详细路线图](docs/roadmap.zh-CN.md)
- [路线图（根目录）](ROADMAP.zh-CN.md)
- [AI 项目上下文提示词](docs/ai-project-context-prompt.zh-CN.md)

## 许可证

LocalBridge 使用 [MIT License](LICENSE) 发布。中文说明见 [LICENSE.zh-CN.md](LICENSE.zh-CN.md)。
