# Sprint 2.8 — TLS Transport Baseline and Paired Certificate Pinning

Status: delivered

## Scope

This sprint establishes the first secure transport contract while preserving explicit HTTP legacy
compatibility. TLS is opt-in because existing installations and Shortcuts use HTTP today.

## Delivered

- `server.tls_enabled`, `server.tls_cert_file` and `server.tls_key_file`, with pair validation in
  application initialization, TLS 1.2 minimum and HTTPS-only server startup.
- Lowercase SHA-256 of the leaf certificate DER in system capabilities and UDP discovery, alongside
  an explicit `transport.https` capability when enabled.
- Pairing/Peer/registry v3 secure metadata and v1/v2 migration to explicit legacy HTTP.
- HTTPS outbound transport pins the exact paired leaf certificate, accepts self-signed certificates
  only through that exact `VerifyConnection`, rejects redirects, and never downgrades secure peers.
- Scheme-aware local GUI, Explorer reuse, capability and file-share URL generation.
- Tests for HTTPS success, wrong fingerprints, HTTP legacy, redirects, missing configuration,
  registry migration, discovery and capabilities.

## Deferred

The product does not yet provide first-use fingerprint confirmation/distribution UX, automatic
certificate rotation, or an iPhone certificate/private-CA installation flow. A self-signed
certificate may work for pinned peer transport while still failing in Windows/iPhone browsers;
install the private CA/certificate in the platform trust store for those clients.

## Checks

Run `gofmt`, `go test ./...`, `go vet ./...`, Windows build, Linux cross-build and `git diff --check`.
