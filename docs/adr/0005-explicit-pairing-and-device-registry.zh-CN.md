# ADR 0005：显式配对与持久化设备注册表

[English version](0005-explicit-pairing-and-device-registry.md)

## 状态

Phase 2 Sprint 2.2 已接受。

## 决策

在 `device.registry_path` 下维护显式配对对端的本地注册表。设备不能仅因为被局域网发现就获得信任。配对必须使用
本地配置的 pairing code，并记录对端 ID、显示名称、地址、端口、能力、状态和时间戳。

配对响应只返回一次生成的 peer Token。设备列表和详情不会返回 peer Token，日志也不会包含 Token。再次配对同一个设备
ID 会轮换 Token；删除对端会撤销本地记录。

## 理由

发现只是未经认证的可达性提示，不是授权。将两者分开可以避免任意局域网设备通过广播数据包就成为可信设备。本 Sprint
使用小型 JSON 注册表即可满足需求，同时为历史/队列出现后的加密存储、迁移和 SQLite 保留存储边界。

## 影响

- v0.1 部署保持兼容；只有配对对端时才会创建注册表。
- pairing code 和 Bearer Token 当前需要手动配置，属于过渡方案。
- 注册表包含未来出站传输需要的 peer Token，必须由宿主账号和文件权限保护。
- 启用认证后，HTTP 认证边界现在会接受 peer Token。自动配置、Token 轮换策略、静态加密存储和设备发现仍属于后续工作。
