# Sprint 2.7 — 配置 Schema 与安全诊断

[English version](sprint-2.7.md)

## 目标

为 LocalBridge 配置建立明确的升级契约，并在不泄漏凭据的前提下诊断生效值。

## 已交付

- 开发和示例配置增加根级 `version: 1`。
- 无版本/版本 0 的 YAML 与 JSON 只在内存迁移，不修改源文件。
- 拒绝负数/未来版本、未知 JSON 字段、未知 YAML 根键/section、嵌套 section 和格式错误的根版本。
- `GET /api/v1/system/config` 返回来源/生效版本、迁移状态和人类可读的生效值。
- 凭据采用显式脱敏，只返回 `bearer_token_configured` 和 `pairing_code_configured`；响应不包含密钥或配置文件路径。
- 关闭认证时仅回环可访问；启用时仅管理 Token 可访问；peer Token 被拒绝。
- 增加 `system.config.read` 能力声明，以及解析器和授权边界回归测试。
- 增加 `-check-config` CLI 校验和脱敏输出，不启动 GUI 或 HTTP Server。

## 兼容性

旧配置继续有效，并报告 `source_schema_version: 0`、`schema_version: 1` 和 `migrated: true`。迁移有意保持非破坏性；
运维人员验证配置后可手动添加 `version: 1`，因此仍可回滚。

## Phase 2 剩余工作

TLS/认证传输、凭据库集成、共享限流/重试策略、更丰富的网络诊断/支持包，以及未来 Schema 的显式迁移仍待完成。
