# Sprint 2.6 — Peer Token 生命周期

[English version](sprint-2.6.md)

## 目标

让已配对设备凭据能够跨重启保留、按期失效并安全轮换，同时不通过常规设备 API 泄漏。

## 已交付

- 版本 2 私有设备注册表持久化，包含当前凭据和有边界的临时旧凭据。
- 可配置的默认 30 天 Token 有效期和默认 10 分钟轮换重叠期。
- 显式 Token 轮换接口，以及同 ID 重新配对时的轮换。
- 目标限定的轮换授权：本机回环、管理 Token 或目标对端自己的 Token。
- 公开诊断字段/状态：`token_issued_at`、`token_expires_at`、`token_expired` 和 `repair_required`。
- 版本 1 迁移、重启验证、重叠/过期测试，以及 Windows 下先关闭再替换文件。
- 临时文件写入、刷新、严格模式与替换；配对/轮换/撤销持久化失败时回滚内存修改。

## API 与配置

- `POST /api/v1/devices/{id}/token/rotate` 返回 `{rotated, device, token}`；Token 只显示一次，
  调用方必须立即安全保存。
- `security.peer_token_ttl: 720h`
- `security.token_overlap_ttl: 10m`，必须短于 Token 有效期。
- 能力声明增加 `device.token.rotate`。

## 验收证据

- 从磁盘重新创建设备模块后，已配对 Token 仍可认证。
- 只有在配置的重叠期内新旧 Token 同时有效，随后旧 Token 失效。
- 当前 Token 到期后不能认证，公开状态变为 `token_expired`。
- 公开响应和日志不包含 peer Token 值。
- 其他对端的 Token 不能替目标对端执行轮换。
- 旧注册表迁移到版本 2，缺失凭据的记录变为 `repair_required`。

## Phase 2 剩余工作

静态凭据保护、面向用户的配置流程、TLS/认证通道设计、限流、共享重试策略、诊断/支持包，以及设备注册表之外的
配置迁移仍待完成。

