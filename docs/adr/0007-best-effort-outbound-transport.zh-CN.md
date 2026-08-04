# ADR 0007：持久化同步引擎之前的尽力而为出站传输

[English version](0007-best-effort-outbound-transport.md)

## 状态

Phase 2 Sprint 2.5 已接受，并作为 Phase 3 的过渡方案。

## 决策

使用小型标准库 HTTP 传输客户端完成认证的对端能力检查和剪贴板投递。设备模块订阅 `clipboard.changed`，只把本地事件
转发给声明支持 `clipboard.text.push` 的已配对对端，永远不转发远程事件。

投递是尽力而为，并限制响应/请求大小和超时。不承诺持久化、重试、排序、回执或离线重放；这些保证属于未来的同步引擎
和任务存储。

## 理由

这样可以得到真实的多主机路径，并验证 peer Token 与能力协商，而无需在功能模块中再实现一套队列。显式检查 source 是
两个 LocalBridge 主机之间最低限度的反馈环保护。

## 影响

- 当两个对端可达且已配对时，LocalBridge 之间可以同步剪贴板。
- 请求失败会出现在日志中，但不会自动重放。
- iPhone Shortcuts 和其他只支持 Pull 的客户端不受影响。
- Phase 3 必须用持久化、幂等的同步/任务流水线替代直接事件转发。
