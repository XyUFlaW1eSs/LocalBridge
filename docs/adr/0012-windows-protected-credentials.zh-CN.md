# ADR 0012：Windows 受保护凭据与注册表迁移

[English](0012-windows-protected-credentials.md)

- 状态：已接受
- 日期：2026-09-17

## 背景

管理 Bearer Token、配对码和 peer Token 轮换状态此前依赖明文 YAML 或注册表 JSON。文件权限可减少意外泄露，
但无法把复制出的秘密绑定到 Windows 用户。LocalBridge 还需要一种迁移：保护或持久化失败时不能破坏仍可用的 v3 注册表。

## 决策

1. `internal/credentials.Protector` 是平台边界。Windows 使用当前用户 DPAPI、`CRYPTPROTECT_UI_FORBIDDEN`、
   以 purpose 字节作为可选 entropy，并用 `LocalFree` 释放返回内存。
2. 凭据存储为 JSON v1，最大 1 MiB，名称格式固定且严格；拒绝未知字段/版本与重复条目，只保存 base64 编码的 DPAPI 密文。
   base64 是传输编码，不是加密。
3. 写入使用 `0600` 临时文件，刷盘后原子替换目标。全部保护操作在替换前完成；失败时旧文件不变。
4. 配置支持 `credential_protection: auto|required|disabled`、存储路径以及 bearer/pairing 引用。内联值与引用互斥；
   `required` 拒绝内联值。Windows `auto` 保护存储和 peer 注册表，同时兼容旧内联配置。
5. CLI 只接受动作和经过验证的引用名。秘密来自隐藏终端输入或 stdin，绝不输出；凭据操作在服务组合前退出。
6. 设备注册表 v4 声明保护提供方。受保护模式下，当前与上一代 peer Token 使用不同的 peer/slot purpose，只允许密文字段。
   v3 明文迁移会在原子替换原文件前加密全部 Token。密文损坏、purpose/provider 不匹配和降级明文都会关闭式失败。
7. 非 Windows 生产保护明确不受支持。只有显式 `disabled` 才允许旧式明文持久化。诊断显示模式、生效保护与秘密来源类别，
   但不包含值、引用名或存储路径。

## 后果

- DPAPI 密文绑定 Windows 用户配置文件。把文件移动到其他账户或主机后无法解密，必须重新配置/配对。
- 不会改写或删除现有 YAML 内联值。运维者应先配置引用并验证，再手动删除旧明文。
- 受保护注册表 v4 不能降级为 `disabled`；v4 `disabled` 注册表可以原子升级到受保护模式。
- macOS Keychain、Linux Secret Service、证书生命周期 UX 和云端密钥服务不在范围内。
