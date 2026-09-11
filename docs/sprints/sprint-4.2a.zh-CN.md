# Sprint 4.2A — 内嵌 Web GUI 与浏览器安全分享

## 目标

为局域网文件分享提供可用的本地浏览器界面，不暴露客户端本地路径，并把二维码和设置契约做成真实、可测试的功能。

## 用户场景

用户打开 `http://127.0.0.1:8899/app/`，选择或拖入多个文件，一次创建一条分享。页面显示分享 URL、真实生成的二维码 PNG、
每个文件的元数据、接收记录和可恢复上传进度。设置可以编辑、恢复默认值，并在重启后保留。

## 已交付范围

- Go 内嵌、无外部 CDN 依赖的 `/app/` 界面，包含分享、接收记录和设置区域。
- 原生文件选择、多选、拖放和安全 multipart 浏览器上传。
- 一个 multipart 批次只创建一条持久化分享；`Idempotency-Key` 保护浏览器重试。
- 浏览器上传写入配置的 `files.share_dir` 下；不接受也不返回客户端本地路径。
- `/api/v1/files/shares/{id}/qr.png` 返回本地生成的真实 `image/png`；JSON `/qr` 继续保留。
- 版本化、原子写入、`0600` 权限的非敏感设置 JSON，以及默认值/重置接口。
- GUI 展示请求 ID 和可读 API 错误。

## 非目标与边界

Sprint 4.2A 不实现 Windows 开机启动注册、托盘/最小化行为、Explorer 右键菜单、原生通知或声音播放。这些设置作为向后兼容
契约保存，实际系统效果推迟到 Sprint 4.2B；当前不宣称已经产生 OS 效果。服务仍是纯 HTTP、面向局域网，不能暴露到公网。

## 公共契约

- `GET /app/`、`/app/app.js`、`/app/styles.css`：内嵌静态 GUI 资源。
- `POST /api/v1/files/browser-shares`：重复的 `files` multipart part，每个批次一条分享。
- `GET /api/v1/files/shares/{id}/qr.png`：256×256 本地 PNG 二维码。
- `GET|PUT /api/v1/settings`、`POST /api/v1/settings/reset`：版本化设置文档。

## 安全与隐私

未启用认证时，管理路由仍只接受回环请求；启用认证后，本地 GUI 可以使用回环请求，局域网管理请求仍需 Bearer 或已配对 peer
Token。multipart 文件名会被收敛为安全 basename，文件模块继续执行配额和过期策略，上传存储位置由服务端配置控制。设置不包含凭据
或 Token。

## 验证

新增测试覆盖内嵌资源、路径遍历拒绝、二维码 PNG 签名、浏览器多文件单分享、本地路径不泄漏、幂等恢复、远程管理拒绝、设置持久化/重置以及
接收页面可恢复性。发布门禁还运行 `go test ./...`、`go vet ./...`、裁剪后的 `go build`、内嵌脚本 `node --check` 和 `git diff --check`。

## 已知限制与后续

GUI 有意保持为本地 Web 界面，而非原生 Windows Shell。Windows 原生系统效果、更完整的传输取消/状态 API，以及 URL/图片/通知模块留待后续工作。
