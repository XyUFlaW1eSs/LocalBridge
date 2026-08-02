# Architecture

[简体中文](architecture.zh-CN.md)

## Goals

LocalBridge is designed as a long-lived, local-first platform rather than a one-off clipboard
script. The core owns process lifecycle, configuration, logging, HTTP transport and module
registration. Features live behind module interfaces and communicate through events.

## Runtime topology

```text
                         Trusted LAN
┌──────────────────┐       HTTP/JSON       ┌─────────────────────────┐
│ iPhone Shortcuts │ ────────────────────> │ LocalBridge Windows     │
│ Push / Pull      │ <──────────────────── │ net/http + EventBus     │
└──────────────────┘                      │ clipboard module        │
                                          └───────────┬─────────────┘
                                                      │ Win32 adapter
                                          ┌───────────▼─────────────┐
                                          │ Windows user clipboard  │
                                          └─────────────────────────┘
```

The iPhone does not run a persistent listener. Push and Pull are explicit user-triggered
Shortcuts, which is compatible with iOS's background execution constraints.

## Package boundaries

- `cmd/localbridge`: flags, signals and process exit codes only.
- `internal/app`: dependency composition; it is the only package that wires concrete modules.
- `internal/config`: validated configuration, no environment-specific behavior in modules.
- `internal/server`: stable HTTP server and system endpoints.
- `internal/module`: lifecycle and route contract for pluggable features.
- `internal/eventbus`: in-process decoupling. Subscribers must tolerate dropped events when
  their buffer is full; events are notifications, not a durable queue.
- `internal/modules/clipboard`: domain behavior and platform interface. Win32 code is isolated
  in a build-tagged adapter.

## Lifecycle

```text
Load config
  -> validate
  -> create logger/EventBus/ModuleManager
  -> register modules and routes
  -> start module watchers
  -> start HTTP server
  -> wait for SIGINT/SIGTERM
  -> stop HTTP server
  -> stop modules in reverse order
```

## Clipboard flow

Remote Push validates JSON, calculates a hash when needed, drops duplicate content, writes the
Windows clipboard and publishes `clipboard.changed`. The watcher sees the Win32 change but
uses a short-lived suppression map to avoid echoing the same content back into the event path.

Local changes follow the inverse path: read text, hash it, store it as `latest`, and publish an
event. Phase 1 does not persist history; the latest item is held in memory.

## Extension rule

New modules should implement `module.Module`, register only their own `/api/v1/<module>`
routes, and publish/subscribe through EventBus rather than taking direct dependencies on
other feature modules. Public contracts belong in `docs/protocol.md` before implementation.
