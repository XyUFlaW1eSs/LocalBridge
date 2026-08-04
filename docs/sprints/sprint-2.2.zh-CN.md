# Sprint 2.2 — 显式配对与设备注册表

[English version](sprint-2.2.md)

## 目标

引入显式信任记录，但不把局域网发现误当成授权。

## 范围

- 带版本的 JSON 本地对端注册表。
- 设备配对、列表、详情和撤销接口。
- pairing code 校验、peer Token 生成，以及响应/日志中的 Token 脱敏。
- 设备模块注册和能力元数据。

## 非目标

- 自动发现、对端出站投递或 peer Token 认证。
- 注册表静态加密或自动 Token 轮换策略。
- iPhone 或其他客户端的原生配对界面。

## 验收标准

- 没有注册表文件时应用仍可启动，只有配对成功后才创建文件。
- 错误 pairing code 和非法设备身份会被拒绝。
- 配对会创建 peer Token，但 list/get 响应和日志不会暴露 Token。
- 再次配对会轮换 Token；删除对端会移除信任记录。
- 注册表写入有大小边界、版本号和严格文件权限。
- 现有剪贴板行为与 v0.1 配置保持兼容。

## 验证

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## 后续工作

Sprint 2.3 应增加作为可达性提示的发现机制、设备健康/最后出现时间更新，以及基于注册表的第一版出站认证传输。
