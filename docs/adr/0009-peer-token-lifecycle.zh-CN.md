# ADR 0009：持久化 Peer Token 生命周期

[English version](0009-peer-token-lifecycle.md)

## 状态

Phase 2 Sprint 2.6 已接受。

## 背景

配对会生成随机 peer Token，但版本 1 注册表序列化的是公开 `Peer` 结构，而 Token 字段为避免 API 泄漏被排除在
JSON 之外。因此进程重启会静默丢失凭据；Token 也没有签发时间、过期时间和有边界的轮换重叠期。

## 决策

- 注册表版本 2 使用私有持久化模型，将当前凭据和临时旧凭据与所有公开 API 结构彻底分离。
- Token 在 `security.peer_token_ttl` 内有效（默认 720 小时）。轮换时，尚未过期的旧 Token 最多保留
  `security.token_overlap_ttl`（默认 10 分钟），且不能超过旧 Token 自身过期时间。
- 使用同一设备 ID 再次配对会轮换 Token；`POST /api/v1/devices/{id}/token/rotate` 提供显式轮换，
  新 Token 只在本次响应中返回一次。
- 远程轮换只允许管理 Token 或目标设备自己的当前/重叠期 Token；本机回环管理也允许。其他已配对设备的 Token
  不能替目标设备轮换。
- 公开列表/详情只暴露签发和过期时间，绝不返回凭据值。
- 版本 1 注册表会自动迁移。由于旧版本从未持久化 Token，受影响对端会变为 `repair_required`，必须重新配对，
  或由本机管理接口轮换。
- 注册表替换前先将临时文件完整写入、刷新并设置严格文件模式，再进行替换。

## 影响

Token 现在能够跨重启保留并按期失效；短重叠期避免轮换瞬间打断正在进行的请求，同时不会无限接受旧凭据。
注册表属于敏感文件：当前 Token 仍以明文 JSON 保存，而且 Windows 文件模式不能替代 ACL 保证，因此数据目录必须只对
当前用户开放。操作系统凭据库集成和认证传输仍属于后续安全加固。

