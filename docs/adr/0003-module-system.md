# ADR 0003: In-process modules and EventBus

[简体中文](0003-module-system.zh-CN.md)

## Decision

Modules own their routes and lifecycle. Cross-feature notifications use a non-blocking in-process
EventBus. Events are ephemeral; durable state belongs to a future storage module.

## Rationale

The architecture supports future file, image and notification features without making the core
depend on every feature, while keeping Sprint 1 small and easy to run.
