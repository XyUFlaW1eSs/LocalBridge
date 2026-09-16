# 更新日志

这里记录 LocalBridge 的重要变更。

## [Unreleased]

- 增加中英文安全威胁模型和强制 v1.0 安全发布门禁。
- Phase 2 Sprint 2.1：增加请求 ID、能力发现和可选 Bearer Token 认证。
- 增加安全配置、部署说明以及过渡安全边界的协议/ADR 文档。
- Phase 2 Sprint 2.2：增加显式配对、peer Token 生成、持久化设备注册表以及撤销/列表接口。
- Phase 2 Sprint 2.3：增加可选 UDP 发现，仅作为不可信的可达性提示。
- Phase 2 Sprint 2.4：启用认证后，已配对 peer Token 可以访问受保护 API。
- Phase 2 Sprint 2.5：增加对端健康探测，以及面向已配对 LocalBridge 对端的尽力而为剪贴板出站投递。
- Phase 2 Sprint 2.6：修复 peer Token 跨重启持久化，增加可配置过期时间、有边界重叠轮换、目标限定轮换授权和版本 1 注册表迁移。
- Phase 2 Sprint 2.7：增加配置 Schema v1、非破坏性旧配置迁移、未来/未知输入严格拒绝、仅管理方可读的脱敏生效配置诊断，
  以及非交互式 `-check-config` 命令。
- Phase 2 Sprint 2.8：增加可选 TLS 1.2+ 和仅 HTTPS 启动、小写叶证书 SHA-256 声明、v3 安全/legacy 对端元数据与 v2 注册表迁移、
  精确证书固定的 HTTPS 对端传输、重定向拒绝以及按 scheme 生成 GUI/Explorer URL。首次指纹 UX、自动证书轮换和 iPhone 信任安装仍是后续工作。
- Phase 2 Sprint 2.9：增加 Windows 当前用户 DPAPI、严格版本化凭据存储、引用式管理/配对秘密、安全 CLI set/delete/status、
  注册表 v4 受保护 peer Token，以及关闭式失败的 v3 明文原子迁移。非 Windows 受保护存储和旧 YAML 内联值自动清理仍是后续工作。
- 增加 `-support-bundle`，生成不覆盖、仅所有者可读的 ZIP；只包含脱敏配置、运行时信息和状态文件元数据，排除凭据、身份、本地路径、
  TLS 材料、载荷和持久化状态正文。
- Phase 3 Sprint 3.1：增加通用 Envelope、持久化同步 Job、保留边界和只读任务查询接口。
- Phase 4 Sprint 4.2A：增加内嵌 `/app/` GUI、浏览器安全多文件分享、接收记录界面、本地真实二维码 PNG 生成和版本化设置持久化。
- Phase 4 Sprint 4.2B 基础：增加归属分享文件清理、当前用户级 Windows 开机启动与 Explorer 右键菜单同步、原生托盘、
  多文件 `-share` 入口、仅元数据的传输事件和完成通知。原生宿主窗口关闭最小化到托盘仍待实现。
- Phase 4 Sprint 4.2C：增加持久化的等待/接收中/已拒绝上传状态、仅回环或已认证可用的允许/拒绝接口、Windows
  接收请求通知，以及手机端等待批准并继续上传的流程。
- Phase 4 Sprint 4.2D：增加纯 Go WebView2 Windows 宿主、真实关闭到托盘、托盘恢复、浏览器回退，以及窗口/健康/关闭/恢复交互式冒烟测试。

## [0.1.0] - 2026-08-02

- 增加基于局域网 HTTP 的 Windows/iPhone 剪贴板 MVP。
- 增加模块化 Go 应用核心、配置、日志、EventBus 与健康检查 API。
- 增加开发文档和快捷指令配置说明。

[English version](CHANGELOG.md)
