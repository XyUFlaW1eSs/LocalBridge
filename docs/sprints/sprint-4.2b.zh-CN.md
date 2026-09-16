# Sprint 4.2B — Windows 外壳基础

[English](sprint-4.2b.md)

## 状态

基础已交付。接收确认随后由 Sprint 4.2C 交付，原生宿主由 Sprint 4.2D 交付。

## 目标

把持久化桌面设置和文件传输事件连接到真实的当前用户级 Windows 效果，同时不要求管理员权限，也不让功能模块直接耦合 Win32。

## 已交付

- `internal/native` 使用构建标签隔离 Windows 代码，其他平台使用空操作适配器。
- `auto_start` 同步 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\LocalBridge`。
- `explorer_context_menu` 在 `HKCU\Software\Classes\*\shell` 下同步当前用户级 `LocalBridgeShare` 动词、显示名、图标、
  多选策略和命令。
- `localbridge.exe -share <file> [file...]` 校验互不重复的普通非符号链接文件，优先复用现有回环服务，并把一次选择创建为一条分享。
- 原生 Windows 通知区域图标提供“文件分享、接收记录、设置、退出”；退出请求走正常应用关闭流程，不直接终止进程。
- 文件创建/完成通过 EventBus 发布仅含元数据的 `file.sent` 与 `file.received`。Windows 适配器显示完成通知，并按总开关与方向开关
  播放提示音，不暴露内容、Token 或路径。
- 浏览器归属的分享文件在逐条删除、全部清除和过期时回收；原生路径分享绝不会删除用户源文件，重启会清除遗留浏览器临时文件。

## 安全边界

- 注册表写入仅限当前用户，并安全引用可执行文件和配置路径。
- Explorer 参数会规范化并使用 `Lstat` 检查；目录、不存在文件、符号链接、重复项和空选择均被拒绝。
- Explorer 命令优先使用回环管理 API；远程文件路径管理在未认证时仍被拒绝。
- 事件载荷只包含标识、数量和大小。

## 验证

- fake registry 测试覆盖启用/禁用、显示名、图标、多选和命令引用。
- 路径测试覆盖普通文件、不存在文件、目录、重复项，以及宿主允许时的符号链接。
- 文件测试覆盖发送/接收事件只发布一次，以及幂等分享重试。
- `go test ./...`、`go vet ./...`、裁剪 Windows 构建和 `git diff --check` 通过。
- Linux 包使用 `go test -c` 做编译检查，Linux 命令通过空操作原生适配器构建。

## 已知限制与下一切片

本基础最初由系统浏览器打开内嵌页面。Sprint 4.2C 随后交付接收确认，Sprint 4.2D 交付 WebView2 宿主与关闭/恢复交互式证据。
注册表行为已有确定性 fake-registry 测试；发布验证仍应在目标安装镜像上实际操作 Explorer。
