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
│ iPhone / Browser  │ ───────────────────> │ LocalBridge Windows     │
│ share / receive   │ <─────────────────── │ GUI + HTTP + EventBus   │
└──────────────────┘                      │ clipboard + files       │
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
- `internal/modules/device`: local device identity, explicit pairing and the versioned persisted
  peer registry. Public models exclude credentials; the private registry stores issue/expiry and
  bounded rotation-overlap state. LAN discovery remains a reachability hint and never grants trust.
- `internal/transport`: bounded HTTP JSON client used for peer capabilities, health checks and
  best-effort outbound delivery. Durable queues and retry policy belong to the future sync engine.
- `internal/modules/clipboard`: domain behavior and platform interface. Win32 code is isolated
  in a build-tagged adapter.
- `internal/modules/files`: persisted share/receive metadata, capability URLs, safe source-file
  inspection, HTTP Range downloads and Content-Range uploads. It owns generated receive paths and
  does not depend on the clipboard module or GUI layer.
- `internal/modules/settings`: versioned atomic JSON settings for non-secret GUI and native preferences.
- `internal/native`: build-tagged platform integration. Windows synchronizes HKCU startup and
  Explorer keys, hosts the tray/message loop, opens the local GUI, and consumes file completion
  events for bounded notifications. Other platforms compile a no-op adapter.
- `internal/web`: embedded vanilla HTML/CSS/JS at `/app/`, with no CDN or network asset dependency.

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

## Web GUI flow

The local GUI is served from embedded resources at `/app/`. Browser file selection and drag/drop
use `POST /api/v1/files/browser-shares` as multipart form data. The server writes each part into
the configured controlled share directory, then creates one share record for the complete batch;
the browser never receives a local path. Share rows fetch the JSON QR contract and PNG from the
management API. Receive history combines active upload offsets with completed receive records.
Settings are read and written through `/api/v1/settings`; local loopback requests are allowed for
the GUI while remote management still requires authentication.

## Windows shell flow

The settings store notifies `internal/native` after a successful durable write. On Windows the
adapter synchronizes only current-user registry keys; it never requires elevation. Explorer invokes
the executable with `-share` and one or more file arguments. The command validates regular files,
prefers the already-running loopback API, and otherwise starts the application and creates one
multi-file share. File modules publish metadata-only transfer events; the native adapter shows a
tray notification and applies the configured sound policy. On Windows a pure-Go WebView2 host
navigates only to the local GUI origin. Its window procedure converts close into hide when
`minimize_to_tray` is enabled; tray actions restore the same window. Missing WebView2 falls back
to the system browser.

## Device credential flow

Pairing creates a random 256-bit peer token and stores it only in the private registry model. Public
device responses contain lifecycle timestamps, not credential values. A rotation installs a new
token and retains the still-valid previous token for a short configured overlap. Validation accepts
only unexpired current/overlap credentials. Expired peers are excluded from health probes and
outbound forwarding. Registry version 1 migrates to version 2; entries affected by the former
non-persistence bug are marked `repair_required` rather than silently trusted.

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
