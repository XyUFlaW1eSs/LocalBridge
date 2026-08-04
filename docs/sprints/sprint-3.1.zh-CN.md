# Sprint 3.1 — 通用 Envelope 与持久化任务状态

[English version](sprint-3.1.md)

## 目标

在增加更多载荷模块前，为跨设备投递提供可解释、可持久化的状态边界。

## 范围

- 带版本的 `Envelope` 和 Job 模型。
- 支持保留策略和状态校验的有边界 JSON 任务存储。
- 只读任务查询接口。
- 集成现有尽力而为剪贴板出站投递。

## 非目标

- 自动重试、离线重放、排序、回执或冲突解决。
- 文件/图片/URL/通知载荷模块。
- 载荷静态加密或 SQLite 迁移。

## 验收标准

- 本地剪贴板投递会创建 pending Job，并记录 delivering/delivered 或 failed。
- Job 状态跨重启保留，非法状态转换会被拒绝。
- 载荷、文件大小、数量和保留限制生效。
- 可以查询 Job，但客户端不能伪造投递状态。
- 现有剪贴板 API 和对端反馈环行为保持不变。

## 验证

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## 后续工作

Sprint 3.2 应先增加重试/离线策略、回执和冲突/幂等测试，再实现富剪贴板格式。
