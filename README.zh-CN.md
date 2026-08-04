# LocalBridge

语言： [English](README.md) · [简体中文](README.zh-CN.md)

LocalBridge 是一个轻量、以本地网络为优先的局域网协作平台。当前第一个可用模块通过两个 iPhone
快捷指令，在 Windows 电脑与 iPhone 之间同步文本剪贴板内容。不需要云账号或中继服务。

> 状态：Phase 0 与 Phase 1 / Sprint 1 已交付。Windows 剪贴板集成在 Windows 构建中启用，其余项目代码
> 保持跨平台并可测试。

## 功能概览

```text
iPhone 快捷指令 --HTTP POST--> Windows 上的 LocalBridge --Win32--> Windows 剪贴板
iPhone 快捷指令 <--HTTP GET--- Windows 上的 LocalBridge <--Win32-- Windows 剪贴板
```

- `POST /api/v1/clipboard`：接收来自 iPhone 快捷指令的文本。
- `GET /api/v1/clipboard/latest`：返回 Pull 快捷指令所需的最新内容。
- Windows 监听器轮询 Win32 剪贴板序列号，并发布本地变化。
- 使用 SHA-256 内容哈希避免重复更新和剪贴板反馈循环。
- 通过 EventBus 与模块边界，为未来的文件、图片和通知功能保留扩展空间。

Windows 本地剪贴板变化目前不会主动向 iPhone 发 HTTP 请求。它会更新内存中的 `latest`，然后由 iPhone Pull
快捷指令主动 GET；这是 iOS 后台限制下的 Phase 1 设计。已配对的其他 LocalBridge 桌面主机现在可以通过尽力而为的
出站路径接收本地剪贴板事件；持久化队列和重试语义计划在 Phase 3 实现。

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
| `internal/server` | 标准库 HTTP 服务与健康接口 |
| `internal/modules/clipboard` | 剪贴板 API、去重和平台适配器 |
| `docs` | 架构、协议、部署与开发规范 |
| `shortcut` | iPhone 快捷指令配置与请求示例 |

## 安全边界

Phase 1 最初没有认证。Phase 2.1 增加了可选 Bearer Token 认证，但配对、Token 配置/轮换和 TLS 仍待实现。服务
仍只适用于可信的家庭/办公局域网，并应将 Windows 防火墙限制在 Private 网络配置文件；不要将 8899 端口暴露到公网。

## 文档

- [架构](docs/architecture.zh-CN.md)
- [协议与 API](docs/protocol.zh-CN.md)
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
- [产品规划与能力地图](docs/product-plan.zh-CN.md)
- [详细路线图](docs/roadmap.zh-CN.md)
- [路线图（根目录）](ROADMAP.zh-CN.md)
- [AI 项目上下文提示词](docs/ai-project-context-prompt.zh-CN.md)

## 许可证

LocalBridge 使用 [MIT License](LICENSE) 发布。中文说明见 [LICENSE.zh-CN.md](LICENSE.zh-CN.md)。
