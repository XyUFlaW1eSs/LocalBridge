# ADR 0001：模块化内部目录

[English version](0001-project-layout.md)

## 决策

使用 `cmd/`、`internal/app`、`internal/server`、`internal/eventbus`、`internal/module` 和
`internal/modules/<feature>`。平台特定的适配器放在对应功能包内，并通过小型接口隔离。

## 理由

这样可以明确可执行程序的组合方式，避免功能包演变为第二套框架，并让未来的文件/图片/设备模块独立于核心。
