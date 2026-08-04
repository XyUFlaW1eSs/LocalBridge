# AI 项目上下文提示词

[English version](ai-project-context-prompt.md)

当需要让其他编码模型理解或继续开发 LocalBridge 时，可以复制下面的提示词。模型仍然必须检查仓库，并以代码和
测试作为实现事实来源。

```text
你正在继续开发 LocalBridge，这是一个位于当前仓库根目录的开源 Go 项目。请先检查仓库，不要假设本提示词一定比代码新。

## 产品定位

LocalBridge 是一个本地优先、模块化、可插拔的局域网跨设备协作平台。它最初是 Windows 与 iPhone 之间的剪贴板桥接器，
但剪贴板只是第一个模块。长期目标是在不依赖云账号或厂商中继的前提下，让可信设备之间传递剪贴板、文件、URL、图片、通知
和安全动作。

## 当前基线

- Phase 0 和 Phase 1 / Sprint 1 已交付。
- 当前版本线是 v0.1.x，已交付包为 v0.1.0。
- 当前可用路径是：可信私有局域网中的 Windows 主机 + iPhone 快捷指令。
- Phase 1 只支持 UTF-8 文本剪贴板。
- Windows 使用 Win32 剪贴板适配器和轮询监听器；其他平台使用安全的空适配器，使核心仍可构建和测试。
- 当前最新剪贴板数据只保存在内存中。
- Phase 2.1 已增加可选 Bearer 认证、请求 ID 和能力发现。
- Phase 2.2 已增加显式配对接口和持久化对端注册表；Phase 2.3 已增加可选 UDP 发现，结果只是临时且不可信的可达性列表；
  Phase 2.4 已让启用认证时的受保护 API 接受已配对 peer Token。TLS、出站对端投递和公共客户端生态尚未实现。
- 当前 Windows 到 iPhone 是 Pull 流程：Windows 更新内存中的 latest，iPhone 必须 GET；在出站传输和对端注册实现前，不要声称支持自动推送。

## 当前 HTTP 契约

- `GET /api/v1/system/health`
- `POST /api/v1/clipboard`：接受包含 `content`、兼容别名 `text`、可选 `type`、`mime_type`、`device_id`、`id`、`hash` 的 JSON 对象；
  也接受原始 UTF-8 `text/plain`。返回 `{accepted, item}`。
- `GET /api/v1/clipboard/latest`：直接返回最新项目；为空时返回 404。
- `GET /api/v1/clipboard/status`：返回 module、enabled 和 has_latest。
- `GET /api/v1/devices`：列出本地设备和已配对对端，不返回 peer Token。
- `GET /api/v1/devices/{id}`：返回单个已配对对端，不返回其 Token。
- `POST /api/v1/devices/pair`：使用配置的 `security.pairing_code` 配对对端，并只在响应中返回一次生成的 peer Token；再次配对会轮换它。
- `DELETE /api/v1/devices/{id}`：撤销对端。
- `GET /api/v1/devices/discovered`：列出临时 UDP 发现提示；发现到的设备不可信，也不会进入已配对注册表。
- Phase 1 默认最大值是 1048576 UTF-8 字节。hash 使用 SHA-256；相同 hash 会被忽略。远程写入有短暂抑制窗口，避免监听器回声。
- 完整契约见 `docs/clipboard.md` 和 `docs/protocol.md`。

## 架构与边界

- `cmd/localbridge`：只负责命令行参数、信号和退出码。
- `internal/app`：依赖组合，连接具体模块。
- `internal/config`：配置加载和校验。
- `internal/server`：标准库 HTTP Server 和系统接口。
- `internal/module`：稳定的模块生命周期和路由契约。
- `internal/eventbus`：进程内非阻塞通知，不是持久队列。
- `internal/modules/device`：显式配对和持久化对端注册表。发现必须只是可达性提示，不能成为授权。
- `internal/modules/clipboard`：剪贴板领域逻辑和平台接口。
- 计划中的分层是：客户端/适配器 -> 身份/信任/发现/策略 -> 传输层 -> 同步引擎 -> 功能模块 -> 存储/可观测性。
- 功能模块不能互相导入或直接控制。使用公共契约和 EventBus 事件。核心负责生命周期、信任、传输和策略；模块负责领域校验和
  平台应用行为。

## 后续开发方向

1. Phase 2 / v0.2.x：配置加固、Token 过期/轮换、TLS、诊断、共享重试/超时、持久化边界和 Windows 服务/托盘设计。
   请求 ID、能力、过渡认证、显式配对、对端注册表、不可信 UDP 发现提示和 peer Token 认证已部分交付。
2. Phase 3 / v0.3.x：通用内容信封、能力协商、出站投递、投递状态、离线队列、历史、富剪贴板、图片、HTML/RTF 和截图。
3. Phase 4 / v0.4.x：支持恢复和完整性校验的文件传输、URL 推送、图片投递、通知、截图及组合上下文任务。
4. Phase 5 / v0.5.x：原生 iOS/iPadOS、Android、macOS、Linux、Windows 托盘/服务、CLI、Web 诊断页和多设备目标选择。
5. Phase 6 / v0.6.x：插件 SDK、Manifest、能力、权限、快捷键、浏览器扩展、Webhook、脚本和安全自动化规则。
6. Phase 7 / v1.0.0：稳定协议策略、安全审查、签名产物、升级/回滚、恢复、性能预算和一致性测试。

完整规划见 `docs/product-plan.zh-CN.md`、`docs/roadmap.zh-CN.md` 及其英文版本。

## 不可违背的工程规则

- 修改前先检查当前代码、测试、git status 和相关文档。
- 保留用户的无关修改；不要使用破坏性的 git 命令。
- 优先小而可审查的变更，避免没有必要的重写。
- 跨模块变化先定义或更新公共契约，再实现行为。
- 功能变化应按需要同步更新代码、测试、文档、配置示例和脚本；中英文文档保持一致。
- 核心工作流保持本地优先；不要悄悄引入云依赖或公共中继。
- 不要把未认证的局域网 API 暴露到公网。
- 不要记录剪贴板/文件/消息正文；只记录大小、hash、ID、来源、状态和错误等元数据。
- 网络载荷必须明确幂等、排序、重试、超时、取消、过期、完整性和重启行为。
- 把操作系统限制作为明确的能力差异，并实现安全回退。
- 除非有明确的维护和安全理由，否则优先使用标准库。

## 开发流程

1. 阅读 `README`、`docs/architecture`、`docs/product-plan`、`docs/roadmap`、`docs/developer-guide`、相关协议文档、ADR 和 Sprint 记录。
2. 检查受影响的包、测试和配置，说明当前行为以及最小可用实现切片。
3. 如果跨越边界，先写设计说明或 ADR，并先记录 API/数据模型。
4. 通过聚焦测试实现，包含异常路径以及重启/网络失败行为。
5. 运行 `gofmt`、`go test ./...`、`go vet ./...`、相关平台构建和 `git diff --check`。
6. 行为变化时同步更新中英文文档、示例、变更日志或发布说明。
7. 汇报变更文件、验证结果、已知限制和下一步安全动作。

## 如何处理具体任务

不要因为未来功能有吸引力就扩大当前任务。如果需求依赖尚不存在的信任、传输或持久化基础设施，应先解释依赖关系，并实现
最小安全前置能力；如果选择会产生实质影响，再请求方向。不能仅因为 HTTP 返回成功就声称功能完成，必须验证端到端用户结果和失败行为。

回复使用以下结构：

1. 当前理解与假设。
2. 计划与受影响边界。
3. 实现摘要。
4. 测试/构建/手工检查。
5. 已知限制与后续工作。
```
