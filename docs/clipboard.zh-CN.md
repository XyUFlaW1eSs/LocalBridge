# 剪贴板模块设计

[English version](clipboard.md)

## 职责

剪贴板模块负责在稳定的 HTTP/EventBus 契约和操作系统剪贴板之间转换。它拥有去重逻辑、内存中的最新项目
以及远程写入抑制窗口。

## 接口

`Platform` 提供 `ReadText`、`WriteText` 和 `Watch`。模块的 `Watcher` 将平台回调适配到领域处理逻辑。
Windows 使用 Win32 Unicode 剪贴板 API 与 `GetClipboardSequenceNumber` 实现；其他操作系统使用安全的空适配器，
确保服务器仍可构建，同时为未来的原生集成保留空间。

## 数据规则

- Sprint 1 只接受 UTF-8 文本。
- 哈希值是 UTF-8 内容的 SHA-256。
- 不论来源设备，只要哈希重复就忽略。
- 远程写入会抑制两秒，避免 `WriteText` 触发本地监听器后形成反馈循环。
- 最新项目只保存在内存中，进程重启后丢失。

## 已知限制

- 监听器采用轮询，以避免在领域层引入 Win32 消息窗口生命周期。
- 当其他 Windows 进程占用剪贴板时，所有权和访问可能暂时失败；监听器会在下一次间隔重试。
- Phase 1 不支持图片、HTML、文件、加密或设备认证。
