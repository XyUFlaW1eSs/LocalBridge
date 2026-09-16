# ADR 0012: Windows protected credentials and registry migration

[简体中文](0012-windows-protected-credentials.zh-CN.md)

- Status: Accepted
- Date: 2026-09-17

## Context

Management bearer tokens, pairing codes and peer-token rotation state previously depended on
plaintext YAML or registry JSON. File permissions reduce accidental disclosure but do not bind a
copied secret to the Windows user. LocalBridge also needs a migration that cannot destroy a usable
v3 registry when protection or persistence fails.

## Decision

1. `internal/credentials.Protector` is the platform boundary. Windows uses current-user DPAPI with
   `CRYPTPROTECT_UI_FORBIDDEN`, purpose bytes as optional entropy, and `LocalFree` for returned memory.
2. The credential store is JSON version 1, bounded to 1 MiB, uses strict fixed-format names, rejects
   unknown fields/versions and duplicates, and stores only base64-encoded DPAPI ciphertext. Base64
   is transport encoding, not encryption.
3. Writes use a `0600` temporary file, flush it, then atomically replace the destination. Protection
   completes before replacement; failures leave the prior file unchanged.
4. Configuration supports `credential_protection: auto|required|disabled`, a store path, and
   bearer/pairing references. Inline and reference values conflict. `required` rejects inline values.
   Windows `auto` protects the store and peer registry while retaining old inline-config compatibility.
5. The CLI accepts only an action and a validated reference name. Secret material comes from hidden
   console input or stdin and is never printed. Credential operations exit before service composition.
6. Device registry v4 declares its protection provider. In protected mode current and previous peer
   tokens use distinct peer/slot purposes and only ciphertext fields are legal. v3 plaintext migration
   encrypts every token before atomically replacing the original. Corrupt ciphertext, purpose mismatch,
   provider mismatch and downgrade to plaintext fail closed. A v4 `disabled` registry may upgrade
   atomically to the configured protector; an already-protected v4 registry may neither downgrade
   to `disabled` nor open under a different provider, and every rejected transition preserves the
   original bytes.
7. Non-Windows production protection is explicitly unsupported. Only explicit `disabled` permits
   legacy plaintext persistence. Diagnostics report mode/effective support and secret source classes,
   but omit values, reference names and store paths.

## Consequences

- DPAPI ciphertext is tied to the Windows user profile. Moving the files to another account or host
  does not preserve decryptability; recovery requires re-provisioning/re-pairing.
- Existing inline YAML is not rewritten or deleted. Operators must provision references and remove
  old plaintext manually after verification.
- Protected registry v4 cannot be downgraded to `disabled`. A v4 `disabled` registry may be upgraded
  to protected mode atomically.
- macOS Keychain, Linux Secret Service, certificate lifecycle UX and cloud key services remain out
  of scope.
