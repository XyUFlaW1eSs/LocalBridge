# Sprint 4.2A — Embedded web GUI and browser-safe sharing

## Goal

Give a local Windows user a usable browser surface for LAN file sharing without exposing local
filesystem paths, while making the QR and settings contracts real and testable.

## User scenario

The user opens `http://127.0.0.1:8899/app/`, selects or drops several files, and creates one share.
The page shows the share URL, a generated QR PNG, per-file metadata, receive records and resumable
upload progress. Settings can be edited, reset to defaults and persisted across restarts.

## Scope delivered

- Go-embedded, dependency-free `/app/` UI with Shares, Receive records and Settings areas.
- Native file picker, multi-select, drag/drop and safe multipart browser upload.
- One multipart batch creates one persisted share; `Idempotency-Key` protects browser retries.
- Browser uploads write below configured `files.share_dir`; client local paths are neither accepted nor
  returned.
- Real local `image/png` QR output at `/api/v1/files/shares/{id}/qr.png`; JSON `/qr` remains available.
- Versioned, atomic, mode-`0600` non-secret settings JSON with defaults and reset endpoint.
- Request IDs and readable API errors are surfaced in the GUI.

## Non-goals and boundaries

Sprint 4.2A does not implement Windows startup registration, tray/minimize behavior, Explorer context
menu integration, native notifications or sound playback. Those settings are stored as a forward-
compatible contract for Sprint 4.2B; they do not claim OS effects yet. The service remains HTTP-only,
LAN-scoped and unsuitable for public exposure.

## Public contracts

- `GET /app/`, `/app/app.js`, `/app/styles.css`: embedded static GUI resources.
- `POST /api/v1/files/browser-shares`: repeated `files` multipart parts, one share per batch.
- `GET /api/v1/files/shares/{id}/qr.png`: 256×256 local PNG QR image.
- `GET|PUT /api/v1/settings`, `POST /api/v1/settings/reset`: versioned settings document.

## Security and privacy

Management routes remain loopback-only without authentication. With authentication enabled, the local
GUI is allowed to use loopback while LAN management requests still need a bearer or paired peer token.
Multipart names are reduced to safe basenames, quotas and expiration are enforced by the files module,
and upload storage is controlled by server configuration. Settings contain no credentials or tokens.

## Verification

The implementation includes tests for embedded assets, traversal rejection, QR PNG signature, browser
multi-file single-share behavior, local-path non-disclosure, idempotent recovery, remote management
rejection, settings persistence/reset and receiver-page resumability. The release gate also runs
`go test ./...`, `go vet ./...`, a stripped `go build`, `node --check` for the embedded script and
`git diff --check`.

## Known limitations and follow-up

The GUI is intentionally a local web surface rather than a native Windows shell. Native OS effects,
broader transfer cancellation/state APIs and URL/image/notification modules remain future work.
