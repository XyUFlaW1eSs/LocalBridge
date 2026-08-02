# LocalBridge

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
| `internal/server` | Standard-library HTTP server and health API |
| `internal/modules/clipboard` | Clipboard API, deduplication and platform adapter |
| `docs` | Architecture, protocol, deployment and development standards |
| `shortcut` | iPhone Shortcut setup and payload examples |

## Security boundary

Phase 1 is deliberately simple: there is no pairing token, authentication or TLS. It is
appropriate only for a trusted home/office LAN, with the Windows firewall restricted to the
private network profile. Pairing and authentication are Phase 2 work; do not expose port 8899
to the public internet.

## Documentation

- [Architecture](docs/architecture.md)
- [Protocol and API](docs/protocol.md)
- [Clipboard module design](docs/clipboard.md)
- [Deployment runbook](docs/deployment.md)
- [Developer guide](docs/developer-guide.md)
- [Coding style](docs/coding-style.md)
- [Sprint 1 delivery](docs/sprints/sprint-1.md)
- [Detailed roadmap](docs/roadmap.md)
- [Roadmap](ROADMAP.md)

## License

LocalBridge is released under the [MIT License](LICENSE).
