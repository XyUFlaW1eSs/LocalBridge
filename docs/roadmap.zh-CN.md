# LocalBridge 路线图

本文档是根目录 [`ROADMAP.md`](../ROADMAP.md) 的详细中文版本。

## Phase 0 — 核心基础（已交付）

- 仓库结构和 Go Module。
- 配置、结构化日志和生命周期管理。
- 带版本健康接口的标准库 HTTP Server。
- 模块接口和非阻塞 EventBus。
- 架构、协议、部署与开发文档。

## Phase 1 — 剪贴板 MVP（已交付）

- Windows 文本剪贴板读写适配器。
- 剪贴板序列号监听器。
- 面向 iPhone 快捷指令的局域网 HTTP Push/Pull API。
- Device ID、SHA-256 去重与反馈循环抑制。
- 测试覆盖和部署/运行手册。

## Phase 2 — 信任与发现

- 配对令牌和设备允许列表。
- 使用 mDNS 或 UDP 广播进行局域网发现。
- 连接诊断和明确的设备管理。

## Phase 3 — 富剪贴板与历史

- 图片和 HTML 格式。
- 有界本地历史记录与冲突策略。
- 通过明确记录边界的存储模块提供可选持久化。

## Phase 4 — 协作模块

- 文件传输、URL 推送、通知和截图。
- Android、macOS 与 Linux 客户端适配器。
- 保留交互式剪贴板访问能力的桌面托盘/服务集成。
