# Sprint 4.2D — Windows 原生宿主

[English](sprint-4.2d.md)

## 状态

已交付。Phase 级发布验证仍需在最终安装镜像上实际检查 Explorer 集成。

## 目标

用真实 Windows 窗口承载现有本地 Web GUI，使关闭到托盘可执行，同时不复制应用状态，也不授予前端直接文件系统访问。

## 已交付设计

- 纯 Go `github.com/jchv/go-webview2` 宿主在锁定的 OS 线程上创建一个 WebView2 窗口，并导航到回环 `/app/` 来源。
- 正常启动打开原生窗口；托盘“文件分享、接受记录、设置”会导航并恢复同一窗口，不会创建多个浏览器标签页。
- 宿主子类化窗口过程。`minimize_to_tray=true` 时，`WM_CLOSE` 隐藏窗口；关闭时则销毁宿主并请求应用正常退出。
- 托盘退出与进程关闭会强制真正关闭宿主，然后正常停止模块和 HTTP。
- WebView2 创建失败时记录有边界错误并打开系统浏览器；服务仍可用，但浏览器窗口无法提供关闭拦截。

## 线程与安全

所有 WebView 调用都通过宿主 UI 线程执行。宿主不暴露 JavaScript 到 Go 的绑定，只加载现有回环 HTTP GUI，因此网络认证与模块 API
仍是安全边界。依赖及其加载器采用 MIT 许可；终端需要 Microsoft Edge WebView2 Runtime，当前 Windows 通常已预装。

## 验证证据

- 完整单元测试、`go vet`、前端语法检查和 Windows 发布构建通过。
- 宿主由 Windows 构建标签隔离，Linux 构建仍可编译。
- 交互式 Windows 冒烟测试使用隔离端口与数据目录：WebView 成功加载 `/app/`，健康状态为 `ok`；发送 `WM_CLOSE` 后窗口隐藏，
  进程和 HTTP 服务仍存活；模拟托盘双击后恢复同一标题窗口。
- 测试后已移除隔离配置，并删除生成的测试文件。

