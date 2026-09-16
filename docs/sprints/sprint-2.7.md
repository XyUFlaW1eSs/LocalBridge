# Sprint 2.7 — Configuration Schema and Safe Diagnostics

[简体中文](sprint-2.7.zh-CN.md)

## Goal

Give LocalBridge configuration an explicit upgrade contract and make effective values diagnosable
without exposing credentials.

## Delivered

- Root configuration schema `version: 1` in development and example files.
- In-memory migration of unversioned/version-0 YAML and JSON without modifying the source file.
- Rejection of negative/future versions, unknown JSON fields, unknown YAML root keys/sections,
  nested sections and malformed root versions.
- `GET /api/v1/system/config` with source/effective versions, migration state and human-readable
  effective values.
- Explicit credential redaction using `bearer_token_configured` and `pairing_code_configured`;
  the response does not contain either secret or the configuration file path.
- Loopback-only access when auth is disabled; management-token-only access when enabled; peer-token denial.
- `system.config.read` capability advertisement and regression tests for parser and authorization
  boundaries.
- `-check-config` CLI validation and redacted output without starting the GUI or HTTP server.

## Compatibility

Legacy files remain valid and report `source_schema_version: 0`, `schema_version: 1` and
`migrated: true`. Migration is intentionally non-destructive. Operators should add `version: 1`
after validating their configuration; rollback therefore remains possible.

## Remaining Phase 2 work

TLS/authenticated transport, credential-vault integration, shared rate/retry policy, richer network
diagnostics/support bundles and explicit migrations for future schema versions remain open.
