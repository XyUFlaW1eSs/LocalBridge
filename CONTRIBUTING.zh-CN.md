# 参与 LocalBridge 贡献

## 工作流程

1. 从 `main` 创建一个职责清晰的分支。
2. 当公共契约变化时，编写或更新设计说明。
3. 每个提交只关注一个问题，最好控制在 300 个变更行以内。
4. 创建 PR 前运行 `go test ./...`、`go vet ./...` 和相关平台构建。
5. 每次功能变更同步更新面向用户的文档与 `CHANGELOG.md`。

提交信息使用 Conventional Commits 前缀，例如 `feat:`、`fix:`、`docs:`、`test:` 和 `chore:`。

## 范围

项目以局域网优先。除非已经记录并评审取舍，否则不要增加云服务、遥测或新的依赖。新能力应作为模块，
通过接口和事件连接到稳定核心。

[English version](CONTRIBUTING.md)
