# Sprint 2.5 — 对端健康与尽力而为剪贴板出站

[English version](sprint-2.5.md)

## 目标

用真实的 LocalBridge-to-LocalBridge 投递验证设备和传输边界，同时明确不提供持久化保证。

## 范围

- 可复用且有边界的 HTTP JSON 传输客户端。
- 使用 `/api/v1/system/capabilities` 的对端健康探测。
- 按能力把本地剪贴板事件转发给已配对对端。
- 抑制 remote 来源事件，避免反馈环。

## 非目标

- 持久化队列、重试调度、投递回执、排序或离线重放。
- iPhone 后台监听器或 iPhone 自动推送。
- TLS、自动凭据配置或原生客户端。

## 验收标准

- 已配对 peer Token 可以认证能力请求。
- 本地剪贴板事件可以到达声明 `clipboard.text.push` 的已配对对端。
- 远程剪贴板事件不会再次转发。
- 对端元数据缺失或能力不支持时可以安全跳过。
- 请求/响应边界和超时可以避免无界资源占用。
- 测试覆盖传输、事件转发、能力过滤和生命周期清理。

## 验证

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## 后续工作

Phase 3 必须先引入通用信封、幂等任务状态、持久化队列、重试、回执和冲突策略，再增加更多载荷类型。
