# Sprint 1 — 剪贴板 MVP

[English version](sprint-1.md)

## 目标

在可信局域网上交付可用的 Windows ↔ iPhone 文本剪贴板交换。

## 交付物

- 带 `Platform` 接口和 Windows Win32 适配器的剪贴板模块。
- HTTP Push、Pull 和 status 接口。
- SHA-256 去重与远程写入循环抑制。
- iPhone 快捷指令配置指南与请求示例。
- 测试/构建检查和部署手册。

## 验收标准

1. `go test ./...` 和 `go vet ./...` 通过。
2. Windows 二进制能够使用示例配置启动。
3. 健康接口返回 `status=ok`。
4. POST 文本返回项目，并能从 latest 接口获取。
5. 重复发送相同内容返回 `accepted=false`。
6. 快捷指令文档足以在没有额外口头说明的情况下完成 Push 和 Pull 配置。

## 明确不在范围内

图片、历史记录、iPhone 自动后台轮询、认证、TLS、发现和 Windows 服务安装。它们已记录在 `ROADMAP.md`。
