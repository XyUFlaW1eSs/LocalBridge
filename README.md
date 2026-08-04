# LocalBridge

语言： [English](README.md) · [简体中文](README.zh-CN.md)

LocalBridge is a lightweight, local-first LAN collaboration platform. Its first usable
module synchronizes text clipboard content between a Windows PC and an iPhone through two
iPhone Shortcuts. No cloud account or relay service is required.

> Status: Phase 0 and Phase 1 / Sprint 1 delivered. The Windows clipboard integration is
> active when built on Windows; the rest of the project remains cross-platform and testable.

## What it does

```text
iPhone Shortcut --HTTP POST--> LocalBridge on Windows --Win32--> Windows Clipboard
iPhone Shortcut <--HTTP GET--- LocalBridge on Windows <--Win32-- Windows Clipboard
```

- `POST /api/v1/clipboard` accepts text from an iPhone Shortcut.
- `GET /api/v1/clipboard/latest` returns the latest item for the Pull Shortcut.
- A Windows watcher polls the Win32 clipboard sequence number and publishes local changes.
- SHA-256 content hashes prevent duplicate updates and clipboard feedback loops.
- EventBus and module boundaries keep future file, image and notification features separate.

Windows clipboard changes are not currently pushed to the iPhone automatically. The watcher
updates the in-memory `latest` item; the iPhone must run the Pull Shortcut to GET it. This is
the Phase 1 design under iOS background execution constraints.

Runtime logs are written to the current console. Use `scripts/run.ps1` to run a release package
in the foreground and see Push, Pull, deduplication and clipboard-write diagnostics. Clipboard
content itself is never logged.

## Quick start on Windows

Requirements: Go 1.24 or newer and a trusted private LAN. The current service intentionally
does not expose authentication or TLS; do not bind it to an untrusted network.

```powershell
Copy-Item configs/config.example.yaml configs/config.yaml
go run ./cmd/localbridge -config configs/config.yaml
```

The default address is `0.0.0.0:8899`. Verify it from the Windows machine:

```powershell
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/health
```

Then replace `WINDOWS_IP` in [`shortcut/README.md`](shortcut/README.md) and create the Push
Clipboard and Pull Clipboard Shortcuts.

## Development

```powershell
go test ./...
go vet ./...
gofmt -w cmd internal
go build ./cmd/localbridge
```

For a local development configuration, use `configs/config.dev.yaml`. A non-Windows build
starts the HTTP service but uses a no-op clipboard integration so core packages can be tested
on CI and on other platforms.

## Repository map

| Path | Responsibility |
| --- | --- |
| `cmd/localbridge` | Process entry point and signal lifecycle |
| `internal/app` | Runtime composition and shutdown |
| `internal/config` | YAML configuration and validation |
| `internal/eventbus` | Typed event names and non-blocking subscriptions |
| `internal/module` | Stable module lifecycle and route registration |
- `internal/modules/device` | Device registry, explicit pairing and peer metadata |
| `internal/server` | Standard-library HTTP server and health API |
| `internal/modules/clipboard` | Clipboard API, deduplication and platform adapter |
| `docs` | Architecture, protocol, deployment and development standards |
| `shortcut` | iPhone Shortcut setup and payload examples |

## Security boundary

Phase 1 started without authentication. Phase 2.1 adds optional Bearer-token authentication,
but pairing, token provisioning/rotation and TLS are still pending. The service remains
appropriate only for a trusted home/office LAN, with the Windows firewall restricted to the
private network profile. Do not expose port 8899 to the public internet.

## Documentation

- [Architecture](docs/architecture.md)
- [Protocol and API](docs/protocol.md)
- [Clipboard module design](docs/clipboard.md)
- [Deployment runbook](docs/deployment.md)
- [Developer guide](docs/developer-guide.md)
- [Coding style](docs/coding-style.md)
- [Sprint 1 delivery](docs/sprints/sprint-1.md)
- [Sprint 2.1 delivery](docs/sprints/sprint-2.1.md)
- [Sprint 2.2 delivery](docs/sprints/sprint-2.2.md)
- [Product plan and capability map](docs/product-plan.md)
- [Detailed roadmap](docs/roadmap.md)
- [Roadmap](ROADMAP.md)
- [AI project context prompt](docs/ai-project-context-prompt.md)

## License

LocalBridge is released under the [MIT License](LICENSE).
