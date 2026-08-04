# Sprint 2.4 — 已配对对端 Token 认证

[English version](sprint-2.4.md)

## 目标

让显式配对签发的 Token 可以被已配对对端使用，同时不暴露注册表密钥或削弱管理 Token 边界。

## 范围

- 基于设备注册表的常量时间 peer Token 校验。
- 启用 `security.auth_enabled` 后，HTTP 认证同时接受管理 Token 和已配对 peer Token。
- 两类 Token 的协议和能力文档。

## 非目标

- 自动配置、Token 过期/轮换策略或注册表静态加密。
- 出站同步、对端健康轮询或完整配对界面。

## 验收标准

- 有效管理 Token 和有效已配对 peer Token 都可以访问受保护接口。
- 无效或已撤销 peer Token 返回 `401`。
- peer Token 使用常量时间比较，且不出现在日志/列表响应中。
- 认证默认关闭，保持 v0.1 兼容性。
- 测试和文档明确 peer Token 只有在启用认证后才生效。

## 验证

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## 后续工作

出站传输和对端健康客户端应复用该边界。v1.0 前要用面向用户的配对流程替代手动管理 Token 和 pairing code 配置。
