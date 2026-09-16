# ADR 0011：TLS 传输基线与已配对叶证书固定

- 状态：已接受
- 日期：2026-09-16
- 范围：Phase 2 Sprint 2.8

## 背景

原有 HTTP 传输需要兼容，但会让局域网观察者看到 Bearer Token 和 peer Token。Discovery 本来就是未认证的，不能作为信任决策。
已配对链路因此需要明确传输模式，以及适用于本地自签名证书的稳定身份。

## 决策

- TLS 位于 `server` 配置区且默认关闭。启用时应用初始化必须加载证书和私钥；监听器只启动 HTTPS，最低 TLS 1.2、最高 TLS 1.3，
  不存在 HTTP 回退。
- 服务端在 capabilities 和 UDP discovery 中声明 `scheme: "https"` 与叶证书 DER 的小写 SHA-256；这些字段只是提示。
- registry v3 和 pairing 携带 `secure`、`scheme`、`certificate_sha256`。`secure: true` 只有在 HTTPS 且指纹严格为 64 位小写十六进制时有效。
  registry v1/v2 数据迁移时明确标记为 legacy HTTP，不伪装成受信任 HTTPS。
- HTTPS 出站使用按对端创建的 TLS 配置，在 `VerifyConnection` 中只接受精确的已配对叶证书指纹，因此可以支持自签名证书，且不使用全局验证绕过。
  所有重定向都会拒绝；secure peer 在 TLS/固定校验失败后绝不重试 HTTP。
- 公开元数据可以显示模式和指纹，但绝不显示 Bearer/peer Token。GUI、Explorer 和能力 URL 按配置 scheme 生成。

## 后果

使用自签名证书时，浏览器、Windows WebView2 和 iPhone 客户端仍需将私有 CA/证书安装到平台信任库。首次指纹确认/分发 UX、自动证书轮换、
以及 iPhone 信任安装仍是后续工作。证书固定不能替代 pairing 流程本身的认证。

## 验证

测试覆盖固定 HTTPS 成功、错误指纹、legacy HTTP、拒绝重定向、TLS 配置缺失、registry v2 迁移、discovery 和 capabilities 字段。
发布前检查包括 `go test ./...`、`go vet ./...`、Windows 原生构建、Linux 交叉构建和 `git diff --check`。
