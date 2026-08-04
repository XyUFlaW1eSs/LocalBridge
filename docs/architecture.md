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
- `internal/modules/device`: local device identity, explicit pairing and the persisted peer
  registry. LAN discovery will remain a reachability hint and must not directly grant trust.
- `internal/transport`: bounded HTTP JSON client used for peer capabilities, health checks and
  best-effort outbound delivery. Durable queues and retry policy belong to the future sync engine.
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

The Phase 1 clipboard module has no outbound HTTP client or peer registry. A Windows clipboard
change is therefore not actively sent to an iPhone; the iPhone Pull Shortcut requests
`GET /api/v1/clipboard/latest`.

## Extension rule

New modules should implement `module.Module`, register only their own `/api/v1/<module>`
routes, and publish/subscribe through EventBus rather than taking direct dependencies on
other feature modules. Public contracts belong in `docs/protocol.md` before implementation.

## Target platform layers

As the roadmap expands, the architecture should converge on these layers:

```text
Clients and OS adapters
        ↓
Device identity, pairing, discovery and policy
        ↓
Transport (HTTP/WebSocket, chunking, retry, integrity)
        ↓
Sync engine (envelope, capabilities, delivery state, conflict policy)
        ↓
Feature modules (clipboard, files, URLs, images, notifications)
        ↓
Storage and observability
```

The layers are intentionally separate. A file module should not implement pairing, and a
client should not need to understand another module's storage. The transport must carry
metadata and delivery state without knowing whether the payload is a clipboard item or a file.
The sync engine owns idempotency, capability fallback and conflict semantics; feature modules
own validation and platform-specific application of their content.

Future plugin support must add explicit manifests, capabilities, configuration namespaces and
permissions. It must not turn arbitrary plugins into unrestricted access to the process,
network or user data.
