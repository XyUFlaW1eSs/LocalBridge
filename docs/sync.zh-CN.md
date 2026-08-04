# 同步引擎基础

[English version](sync.md)

Phase 3 先引入通用信封和持久化任务状态。当前实现包装已有的尽力而为剪贴板出站路径，尚未提供重试、离线重放或冲突解决。

## Envelope

`Envelope` 独立于具体功能模块描述内容/动作传输：

| 字段 | 含义 |
|---|---|
| `id` | 用于幂等的稳定事件/任务 ID。 |
| `kind` | 领域操作，当前为 `clipboard.push`。 |
| `type` | 逻辑类型，当前为 `text`。 |
| `mime_type` | 载荷 MIME 类型，通常为 `text/plain`。 |
| `hash` | 内容去重/完整性 hash。 |
| `source_device_id` | 产生内容的设备。 |
| `target_device_id` | 目标对端（如果是定向投递）。 |
| `correlation_id` | 用于追踪关联工作的来源事件 ID。 |
| `size` | 载荷字节大小。 |
| `payload` | 为未来重放保留的 JSON 载荷，应视为私密数据。 |
| `created_at` | UTC 创建时间。 |
| `expires_at` | 可选过期时间。 |

功能模块校验自己的载荷；同步层负责身份、状态、保留和投递语义。

## 任务状态

```text
pending -> delivering -> delivered
                    \-> failed -> delivering
pending/delivering/failed -> canceled 或 expired
```

每次转换都会更新 `updated_at`，进入 `delivering` 时增加 `attempts`。非法转换会被拒绝。相同 ID 再次提交时会幂等返回
已有的 delivered 任务，不会重复创建。

## 当前 HTTP 查询接口

`GET /api/v1/sync/jobs?limit=100` 返回最近任务，limit 会限制在 `1..1000`。

`GET /api/v1/sync/jobs/{id}` 返回单个任务或 `404`。

这些接口仅用于查询；客户端不能直接把任务标记为 delivered，投递由创建任务的传输/功能模块负责。

## 存储

当前使用带版本的 JSON 文件，路径由 `sync.store_path` 配置（默认 `data/sync-jobs.json`）。文件限制为 16 MiB，同时支持
可配置最大任务数和保留时间。文件以严格权限创建。任务载荷可能包含剪贴板正文，因此必须保护 data 目录并设置合理保留策略。

当历史、队列和并发客户端需要更强持久性时，可以用 SQLite 或其他事务存储替换当前实现；公开边界是存储行为，不是 JSON 文件格式。

## 当前限制

- 只有剪贴板出站投递会创建任务。
- 失败任务会记录，但不会自动重试。
- 没有离线队列重放、投递回执、排序保证或冲突策略。
- 尚未实现静态载荷加密和面向用户的历史删除。
