# ADR 0011: TLS Transport Baseline and Paired Leaf-Certificate Pinning

- Status: Accepted
- Date: 2026-09-16
- Scope: Phase 2 Sprint 2.8

## Context

The original LocalBridge HTTP transport is useful for compatibility but exposes bearer and peer
tokens to any observer on the LAN. Discovery is deliberately unauthenticated and cannot be used
as a trust decision. Paired links therefore need an explicit transport mode and a stable identity
for self-signed local certificates.

## Decision

- TLS is configured explicitly under `server` and is disabled by default. Enabling it requires both
  certificate and private-key files to load during application initialization. The listener starts
  HTTPS only, with TLS 1.2 minimum and TLS 1.3 maximum; there is no HTTP fallback.
- The server advertises `scheme: "https"` and the lowercase SHA-256 of the leaf certificate DER in
  capabilities and UDP discovery. These fields are hints only.
- Registry v3 and pairing carry `secure`, `scheme`, and `certificate_sha256`. `secure: true` is valid
  only with HTTPS and an exact 64-character lowercase hexadecimal fingerprint. Registry v1/v2 data
  is migrated as explicit legacy HTTP, never as trusted HTTPS.
- Outbound HTTPS uses a per-endpoint TLS configuration whose `VerifyConnection` accepts the exact
  paired leaf fingerprint, allowing a self-signed certificate without a global verification bypass.
  It rejects every redirect; secure peers never retry HTTP after a TLS or pinning failure.
- Public metadata may expose mode and fingerprint but never bearer or peer tokens. Generated GUI,
  Explorer and capability URLs use the configured scheme.

## Consequences

Browsers, Windows WebView2 and iPhone clients still need the private CA/certificate installed in
their platform trust stores when a self-signed certificate is used. Initial fingerprint
confirmation/distribution UX, automatic certificate rotation, and iPhone trust installation are
follow-up work. Pinning is not a replacement for authenticating the pairing flow.

## Verification

Tests cover successful pinned HTTPS, wrong fingerprints, legacy HTTP, redirect rejection, missing
TLS configuration, registry v2 migration, discovery and capabilities fields. `go test ./...`,
`go vet ./...`, native Windows build, Linux cross-build and `git diff --check` are release checks.
