# 架构

[English version](architecture.md)

## 目标

LocalBridge 被设计为长期维护的本地优先平台，而不是一次性的剪贴板脚本。核心负责进程生命周期、配置、
日志、HTTP 传输和模块注册；功能通过模块接口提供，并通过事件通信。

## 运行时拓扑

```text
                         可信局域网
┌──────────────────┐       HTTP/JSON       ┌─────────────────────────┐
│ iPhone 快捷指令  │ ────────────────────> │ Windows LocalBridge     │
│ Push / Pull      │ <──────────────────── │ net/http + EventBus     │
└──────────────────┘                      │ 剪贴板 + 文件模块       │
                                          └───────────┬─────────────┘
                                                      │ Win32 适配器
                                          ┌───────────▼─────────────┐
                                          │ Windows 用户剪贴板       │
                                          └─────────────────────────┘
```

iPhone 不运行持久化监听器。Push 和 Pull 是用户主动触发的快捷指令，符合 iOS 的后台执行约束。

## 包边界

- `cmd/localbridge`：只负责参数、信号和进程退出码。
- `internal/app`：依赖组合；只有这里连接具体模块。
- `internal/config`：校验配置，模块中不放环境相关行为。
- `internal/server`：稳定的 HTTP 服务和系统接口。
- `internal/module`：可插拔功能的生命周期与路由契约。
- `internal/eventbus`：进程内解耦。订阅者必须容忍缓冲区满时事件被丢弃；事件是通知，不是持久队列。
- `internal/modules/device`：本地设备身份、显式配对和持久化对端注册表。局域网发现只能作为可达性提示，不能直接授予信任。
- `internal/transport`：用于对端能力、健康检查和尽力而为出站投递的有边界 HTTP JSON 客户端。持久化队列和重试策略属于未来的同步引擎。
- `internal/modules/clipboard`：剪贴板领域行为与平台接口。Win32 代码通过构建标签隔离。
- `internal/modules/files`：持久化分享/接收元数据、能力 URL、安全源文件检查、HTTP Range 下载和
  Content-Range 上传。它负责生成接收路径，不依赖剪贴板模块或 GUI 层。

## 生命周期

```text
加载配置
  -> 校验
  -> 创建 Logger/EventBus/ModuleManager
  -> 注册模块和路由
  -> 启动模块监听器
  -> 启动 HTTP Server
  -> 等待 SIGINT/SIGTERM
  -> 停止 HTTP Server
  -> 按逆序停止模块
```

## 剪贴板流程

远程 Push 校验 JSON，必要时计算哈希，丢弃重复内容，写入 Windows 剪贴板并发布 `clipboard.changed`。
监听器会看到 Win32 变化，但通过短时抑制表避免把同一内容再次回传形成循环。

本地变化走相反路径：读取文本、计算哈希、保存为 `latest` 并发布事件。Phase 1 不持久化历史记录，最新
项目只保存在内存中。

Phase 1 剪贴板模块没有出站 HTTP 客户端或对端设备注册表。因此 Windows 剪贴板变化不会主动发送到 iPhone；
必须由 iPhone Pull 快捷指令请求 `GET /api/v1/clipboard/latest`。

## 扩展规则

新模块应实现 `module.Module`，只注册自己的 `/api/v1/<module>` 路由，并通过 EventBus 发布/订阅，而不是
直接依赖其他功能模块。公共契约应在实现前写入 `docs/protocol.md`。

## 目标平台分层

随着路线图扩展，架构应逐步收敛为以下几层：

```text
客户端与操作系统适配器
        ↓
设备身份、配对、发现与策略
        ↓
传输层（HTTP/WebSocket、分块、重试、完整性）
        ↓
同步引擎（信封、能力、投递状态、冲突策略）
        ↓
功能模块（剪贴板、文件、URL、图片、通知）
        ↓
存储与可观测性
```

这些层必须保持分离。文件模块不应该自己实现配对，客户端也不应该理解另一个模块的存储方式。传输层只负责携带
元数据和投递状态，不应该知道载荷是剪贴板项目还是文件。同步引擎负责幂等、能力回退和冲突语义；功能模块负责
内容校验以及平台相关的应用行为。

未来的插件支持必须有明确的 Manifest、能力声明、配置命名空间和权限模型，不能让任意插件无限制访问进程、网络
或用户数据。
