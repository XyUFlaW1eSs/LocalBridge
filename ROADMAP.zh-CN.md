# 路线图

[English version](ROADMAP.md)

LocalBridge 正从 Windows/iPhone 剪贴板 MVP 演进为本地优先、模块化的局域网跨设备协作平台。

## 已交付

- Phase 0：仓库初始化与核心运行时。
- Phase 1 / Sprint 1：Windows 与 iPhone 快捷指令之间的文本剪贴板交换。
- `v0.1.0`：可运行的 Windows 发布包、前台诊断日志和完整中英文文档。

## 计划版本

- **Phase 2 / `v0.2.x` — 平台加固：** Sprint 2.1 已交付请求 ID、能力发现和过渡认证；剩余工作包括配置版本化、
  配对、认证配置/轮换、设备注册表、局域网发现、诊断、共享重试/超时策略以及服务/托盘设计。
- **Phase 3 / `v0.3.x` — 同步引擎与富剪贴板：** 通用信封、能力协商、出站投递、离线队列、历史、图片、HTML/RTF 和截图。
- **Phase 4 / `v0.4.x` — 局域网协作：** 文件传输、URL 推送、图片投递、通知、组合上下文任务和可恢复传输状态。
  当前先交付多文件分享模型、二维码/HTTP 移动端页面和 Windows GUI 基础。
- **Phase 5 / `v0.5.x` — 客户端与设备：** 原生 iOS/iPadOS、Android、macOS、Linux、Windows 托盘/服务、CLI、Web 诊断页和多设备目标选择。
- **Phase 6 / `v0.6.x` — 自动化与插件：** 插件 SDK、权限、快捷键、浏览器扩展、Webhook、脚本、命令面板和安全工作流规则。
- **Phase 7 / `v1.0.0` — 稳定平台：** 协议兼容策略、安全审查、签名产物、升级/回滚、恢复、性能预算和一致性测试套件。

详细能力地图、头脑风暴清单、依赖、优先级和发布门禁见 [`docs/roadmap.zh-CN.md`](docs/roadmap.zh-CN.md)、
[`docs/product-plan.zh-CN.md`](docs/product-plan.zh-CN.md) 与 [`docs/file-sharing-product-plan.zh-CN.md`](docs/file-sharing-product-plan.zh-CN.md)。
开发流程见 [`docs/developer-guide.zh-CN.md`](docs/developer-guide.zh-CN.md)，子任务协作流程见
[`docs/engineering-workflow.zh-CN.md`](docs/engineering-workflow.zh-CN.md)。
