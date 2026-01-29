# Tabs Implementation Tasks

Last updated: 2026-01-29

## Phase 1: Core Daemon & CLI
- [x] Add `push_session` to unix socket protocol (daemon + CLI) per docs/04-api-design.md
- [x] Implement local session export -> remote upload (build upload payload from JSONL)
- [x] Hook installation should update `~/.claude/config.yaml` and `~/.cursor/hooks.json`
- [x] Add Cursor integration: hook handler + SQLite poller + session capture
- [x] Add unit tests for PID/lock, JSONL writer, socket protocol
- [x] Reduce config parsing duplication (use internal/config in CLI)

## Phase 2: Local UI
- [x] Implement Share modal + `/api/sessions/push` flow
- [x] Add tag input + default tags support (from config)
- [x] Implement TanStack Start SSR app (fix route wiring + remove example route mismatches)
- [x] Add date + cwd filters and server-backed search in local timeline
- [x] Wire default tags into Share modal and Settings page
- [x] Playwright coverage for timeline, session detail, search, share flow

## Phase 3: Remote Server + UI
- [x] API key management endpoints: POST/GET/DELETE /api/keys
- [x] Remote UI: keys page uses API endpoints (create + list + revoke)
- [x] Enforce configurable auth on JSON browse/key endpoints (AUTH_MODE: off/header/iap-google)
- [x] Document auth modes (README + docs/04-api-design.md)
- [x] Pagination + sorting for /api/sessions (page, limit, sort, order)
- [x] Tag filter + search parity with spec
- [x] Add Dockerfile for tabs-server

## Phase 4: Cursor Support
- [x] Implement Cursor hooks + polling per docs/IMPLEMENTATION-READY.md
- [x] Validate Cursor sessions appear locally

## Improvements / Fixes
- [x] Fix daemon lock stale-file edge case
- [x] Accept tool_result content that isn’t string (remote upload)
- [x] Add structured logging and error wrapping
