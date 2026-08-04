# 更新日志

这里记录 LocalBridge 的重要变更。

## [Unreleased]

- Phase 2 Sprint 2.1：增加请求 ID、能力发现和可选 Bearer Token 认证。
- 增加安全配置、部署说明以及过渡安全边界的协议/ADR 文档。
- Phase 2 Sprint 2.2：增加显式配对、peer Token 生成、持久化设备注册表以及撤销/列表接口。
- Phase 2 Sprint 2.3：增加可选 UDP 发现，仅作为不可信的可达性提示。
- Phase 2 Sprint 2.4：启用认证后，已配对 peer Token 可以访问受保护 API。
- Phase 2 Sprint 2.5：增加对端健康探测，以及面向已配对 LocalBridge 对端的尽力而为剪贴板出站投递。

## [0.1.0] - 2026-08-02

- 增加基于局域网 HTTP 的 Windows/iPhone 剪贴板 MVP。
- 增加模块化 Go 应用核心、配置、日志、EventBus 与健康检查 API。
- 增加开发文档和快捷指令配置说明。

[English version](CHANGELOG.md)
