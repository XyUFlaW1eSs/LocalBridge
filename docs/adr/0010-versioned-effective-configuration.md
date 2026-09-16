# ADR 0010: Versioned and Redacted Effective Configuration

[简体中文](0010-versioned-effective-configuration.zh-CN.md)

## Status

Accepted for Phase 2 Sprint 2.7.

## Context

LocalBridge previously applied defaults over an unversioned YAML/JSON document. That preserved
simple compatibility but did not define migration or future-version behavior. Operators also had
no safe way to inspect the effective values after defaults without opening a potentially secret
configuration file.

## Decision

- The current root configuration schema is version 1 and checked-in configurations declare
  `version: 1`.
- A document without `version`, or with explicit version 0, is treated as legacy v0 and migrated
  to v1 in memory. LocalBridge never rewrites the user's file automatically.
- Negative and future versions are rejected. YAML root keys/sections and JSON fields remain strict;
  unknown values fail startup instead of being silently ignored.
- `GET /api/v1/system/config` exposes the effective configuration with durations formatted for
  humans and with credential values replaced by configured/not-configured booleans.
- The response reports effective and source schema versions, whether migration occurred, and only
  a source class (`file`, `defaults`, or `programmatic`). It does not expose the configuration file
  path.
- With authentication disabled, loopback may read the diagnostic and remote LAN requests are
  rejected. With authentication enabled, every request requires the management bearer token;
  a peer token is intentionally insufficient.

## Consequences

Upgrades now have an explicit compatibility boundary and old files continue to start. Future schema
changes must add a reviewed migration before incrementing the current version. The narrow built-in
YAML subset remains intentionally limited and is not a general YAML implementation. Diagnostics
still expose non-secret paths and limits to an authorized manager, so they must not become public.
