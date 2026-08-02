# 部署手册（Windows）

[English version](deployment.md)

## 1. 构建

安装 Go 1.24 或更高版本，并在 Windows 上构建：

```powershell
go test ./...
go vet ./...
go build -trimpath -ldflags "-s -w" -o .\dist\localbridge.exe .\cmd\localbridge
```

## 2. 配置

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
notepad .\configs\config.yaml
```

设置稳定的 `device.id`。局域网使用时保持 `server.host` 为 `0.0.0.0`，并选择未被占用的端口。如果配置以后
包含凭据，不要提交 `configs/config.yaml`。

## 3. 防火墙

仅在 Private 配置文件中允许 TCP 8899 入站；如果修改端口，请同步替换命令中的端口：

```powershell
New-NetFirewallRule -DisplayName "LocalBridge (Private LAN)" `
  -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8899 -Profile Private
```

不要为 Public 配置文件创建规则，也不要从路由器将此服务端口转发到公网。

## 4. 运行与验证

```powershell
.\dist\localbridge.exe -config .\configs\config.yaml
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/health
```

在 iPhone 上将快捷指令 URL 设置为 Windows 私有 IPv4 地址。依次测试 Push、Pull，然后在 Windows 上直接复制
文本，并在等待监听间隔后确认 latest 接口已更新。

## 5. 服务安装

服务管理器集成明确不属于 Sprint 1。首次部署时应在受监管的用户会话中运行程序，因为 Windows 剪贴板访问属于
交互式桌面会话。未来的 Windows 服务/托盘设计必须保留这一会话要求，并记录安全边界。

## 回滚

停止进程，替换为上一版本的可执行文件，并使用相同配置重启。Phase 1 的 latest 项目只保存在内存中，因此回滚
不需要迁移存储。
