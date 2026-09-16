# Security Threat Model

[简体中文](security-threat-model.zh-CN.md)

## Scope and assumptions

LocalBridge is a local-first collaboration service for a trusted private LAN. “Trusted LAN” reduces
exposure; it does not make traffic, discovery packets or every host trustworthy. Port forwarding,
public-cloud exposure and use on hostile public Wi-Fi are outside the supported deployment model.

The operating-system account running LocalBridge is trusted. A fully compromised paired device or
Windows account can access content that the user authorized for it; preventing that is outside the
process boundary, but revocation, bounded retention and observable actions limit damage.

## Assets

- Clipboard text and future rich clipboard formats.
- Shared and received file bytes, names, hashes, paths and transfer history.
- Management bearer token, pairing code, peer tokens and future private keys.
- Device identity, peer registry, capabilities, network addresses and last-seen metadata.
- Settings that enable startup, Explorer integration, automatic receipt and notifications.
- Integrity of actions such as approving an upload, opening a URL or invoking a future plugin.

## Trust boundaries

| Boundary | Trust level | Required control |
|---|---|---|
| Loopback GUI and native shell | Local management | Origin/loopback checks, bounded requests, no secret reflection |
| Authenticated management client | Highest remote privilege | Management-token scope, TLS, audit metadata, no peer-token substitution |
| Paired peer | Content transport only | Per-peer credential, expiry/revocation, capability checks, TLS pinning |
| Public share/receive URL | One capability only | Random token, expiry, path isolation, size/range validation, revocation |
| UDP discovery | Untrusted hint | No credentials, no automatic pairing, bounded parser, manual verification |
| Persistent stores | Sensitive local state | Restricted permissions, atomic writes, retention, migration and backup policy |
| Future plugin/client | Explicitly permissioned | Manifest, least privilege, version contract, approval for sensitive actions |

## Adversaries and abuse cases

1. A passive LAN observer captures clipboard text, file bytes or bearer tokens from HTTP.
2. An active LAN attacker spoofs discovery, redirects a client, substitutes a certificate or forces
   a secure peer back to HTTP.
3. A paired but malicious device uses its token to invoke management-only APIs or inspect another
   peer's credentials/configuration.
4. A leaked capability URL is replayed before expiry or distributed outside the intended LAN.
5. Crafted names, paths, ranges, JSON bodies or oversized responses attempt traversal, overwrite,
   memory exhaustion, disk exhaustion or parser ambiguity.
6. A malicious webpage targets loopback management endpoints through CSRF-like browser behavior.
7. Logs, diagnostics, notifications, history, backups or support bundles disclose content or secrets.
8. Interrupted writes or upgrades corrupt registries and leave credentials/actions in an ambiguous state.
9. A future plugin, webhook or automation rule exceeds its declared permission or creates an action loop.

## Security invariants

- Discovery never grants trust. Pairing or explicit local management is the only trust transition.
- Credentials and payload bodies are never logged, returned by list APIs or included in notifications.
- Management credentials and peer credentials have different authorization scopes.
- A secure peer never falls back to plaintext after TLS, certificate or fingerprint failure.
- HTTP clients do not follow redirects carrying credentials across origin or scheme boundaries.
- Public capability tokens authorize only one bounded share/receive resource and expire.
- File writes stay under owned directories; client filenames never select an arbitrary destination path.
- Upload approval is enforced before bytes when automatic acceptance is disabled.
- Every parser, request, response, file count, byte count, history and retention period has a bound.
- Durable mutations are atomic or rolled back in memory when persistence fails.
- Future plugins and automation are denied access unless a declared permission grants it.

## Implemented controls

- Optional management Bearer authentication, request IDs and structured metadata-only logs.
- Explicit pairing, persisted expiring peer tokens, bounded rotation overlap and revocation.
- Management-only redacted effective configuration diagnostics.
- Untrusted bounded UDP discovery and capability negotiation.
- Random expiring file capability URLs, safe owned paths, Range/Content-Range validation, checksums,
  receive approval and restart-safe transfer state.
- Atomic-style versioned stores with migration checks and bounded retention/counts.
- Windows current-user registry integration without elevation and no payload/path data in events.
- TLS 1.2+ HTTPS-only startup when enabled, exact leaf-certificate pinning, redirect rejection and
  no HTTP fallback for secure peers (Sprint 2.8).
- Non-overwriting redacted support bundles that exclude secrets, identity, paths, TLS material,
  payloads and state bodies.
- Windows current-user DPAPI for credential references and registry v4 peer tokens, with
  purpose-bound ciphertext and fail-closed atomic v3 migration (Sprint 2.9).

## Open risks before v1.0

- TLS remains opt-in for compatibility; deployments that explicitly keep HTTP expose tokens and
  content to an active LAN attacker. Secure peers themselves do not downgrade.
- macOS Keychain and Linux Secret Service are not implemented. Those platforms must explicitly use
  `disabled` plaintext compatibility mode or refuse startup; Windows users may also leave legacy
  inline YAML values until they manually replace them with references.
- Pairing-code provisioning and first certificate-fingerprint verification do not yet have a polished,
  user-verifiable ceremony.
- Shared rate limiting, connection quotas, disk reservation and abusive-peer backoff remain incomplete.
- Browser origin/CSRF policy and a Content Security Policy require a dedicated review.
- There is no signed installer/update channel, dependency SBOM, fuzzing campaign or external audit.
- Privacy-preserving deletion verification and an incident-response/revocation runbook are not yet implemented.

## TLS and pairing acceptance criteria

- TLS configuration is validated before modules start; incomplete or unreadable key material fails closed.
- The server advertises the leaf certificate SHA-256 fingerprint as metadata, never as proof of identity.
- A secure peer persists the verified fingerprint and uses exact certificate pinning on every request.
- Wrong/changed certificates, TLS errors and redirects fail the job; no HTTP retry occurs.
- Legacy HTTP peers are visibly marked and require an explicit re-pair/upgrade action to become secure.
- Certificate renewal requires an authenticated, user-observable pin update with a bounded overlap or
  explicit approval; silently replacing the pin is forbidden.
- iPhone/Windows trust installation and private-CA guidance are documented and manually verified.

## v1.0 security release gate

Before `v1.0.0`, LocalBridge must additionally complete and record:

1. Secure-by-default transport decision and migration from legacy HTTP peers.
2. Credential-at-rest protection or a documented platform-backed equivalent.
3. Per-route rate/size limits, connection and disk quotas, and adversarial tests.
4. CSRF/origin/CSP review for every browser-reachable management mutation.
5. Fuzz tests for discovery, configuration, ranges, upload state and protocol envelopes.
6. Dependency audit/SBOM, signed reproducible artifacts, update rollback and migration rehearsal.
7. A sanitized support bundle, privacy deletion test and incident-response/revocation runbook.
8. Cross-client conformance tests proving authentication, downgrade rejection and capability fallback.
