# Sprint 2.8 — TLS 传输基线与已配对证书固定

状态：已交付

## 范围

本 Sprint 建立首个安全传输契约，同时保留明确的 HTTP legacy 兼容。由于现有安装和 Shortcuts 使用 HTTP，TLS 默认关闭。

## 已交付

- `server.tls_enabled`、`server.tls_cert_file`、`server.tls_key_file`，应用初始化阶段成对校验，最低 TLS 1.2，服务端仅 HTTPS 启动。
- 在系统 capabilities 和 UDP discovery 中声明叶证书 DER 的小写 SHA-256，并在启用时声明明确的 `transport.https` 能力。
- pairing/Peer/registry v3 安全元数据，以及 v1/v2 迁移为明确 legacy HTTP。
- HTTPS 出站传输固定精确的已配对叶证书；只有精确 `VerifyConnection` 才接受自签名证书，拒绝重定向，secure peer 绝不降级。
- 本机 GUI、Explorer 复用、capability 和文件分享 URL 按 scheme 生成。
- `-support-bundle` CLI：不启动模块即可生成不覆盖已有文件的脱敏 ZIP，只包含安全配置/运行时字段和状态文件元数据。
- 覆盖 HTTPS 成功、错误指纹、HTTP legacy、重定向、配置缺失、registry 迁移、discovery 和 capabilities 的测试。

## 延后工作

当前尚未提供首次指纹确认/分发 UX、自动证书轮换或 iPhone 证书/私有 CA 安装流程。自签名证书可能可以用于固定的对端传输，
但仍可能被 Windows/iPhone 浏览器拒绝；这些客户端应安装私有 CA/证书到平台信任库。

## 检查

运行 `gofmt`、`go test ./...`、`go vet ./...`、Windows 构建、Linux 交叉构建和 `git diff --check`。
