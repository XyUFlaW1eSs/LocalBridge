# Sprint 2.9: Windows protected credential storage

[简体中文](sprint-2.9.zh-CN.md)

## Outcome

Windows deployments can keep management bearer tokens, pairing codes and peer-token rotation state
under current-user DPAPI. Configuration references resolve before modules are constructed, registry
v3 plaintext migrates to protected v4 atomically, and every protection/decryption error fails startup
without rewriting the source file.

## Delivered

- Injectable `Protector`, Windows DPAPI implementation and explicit non-Windows unavailable provider.
- Strict, versioned, 1 MiB credential store with validated names, bounded values, duplicate/unknown
  rejection, owner-only temporary writes, flush and atomic replacement.
- `auto`, `required` and `disabled` policy; protected references for management/pairing values;
  inline/reference conflict checks and resolved management-token length validation.
- `-credential-action set|delete|status` plus `-credential-name`; hidden console or stdin input,
  no secret arguments/output, deterministic exit codes and no service startup.
- Registry v4 protected current/previous token slots, purpose binding, restart decryption, v3 migration,
  plaintext-disk exclusion and downgrade/provider/ciphertext failure handling.
- Redacted configuration/support-bundle protection metadata without secret values, reference names,
  credential-store paths or state bodies.
- Unit coverage for DPAPI/store round trips, purpose mismatch, corruption, bounds/version/duplicates,
  non-overwrite failure, YAML/JSON policy, app resolution, CLI behavior, registry migration/rotation,
  restart and support-bundle redaction.

## Compatibility and limitations

- Schema remains v1 because the new fields are additive and defaults preserve Windows legacy config
  meaning. Source YAML is never rewritten.
- Windows `auto` uses DPAPI. `required` additionally rejects inline secrets. Explicit `disabled`
  keeps plaintext compatibility and is visibly reported in diagnostics.
- Non-Windows production protection is unsupported; macOS Keychain and Linux Secret Service are not
  implemented. DPAPI files are not portable across users or machines.
- The Sprint does not remove old inline YAML values, implement certificate trust/rotation UX, or add
  a cloud key service.

## Verification contract

Run `gofmt`, `go test ./...`, `go vet ./...`, `git diff --check`, a Windows build/test and
`GOOS=linux CGO_ENABLED=0 go build ./...`. Migration failure tests must verify original bytes remain
unchanged, and disk assertions must search for both current and previous plaintext tokens.
