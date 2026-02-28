# TGUP Go Migration Roadmap

Status: Draft  
Scope: Full, behavior-preserving migration from the current Python implementation to a new pure Go implementation based on `gotd/td`

## 1. Goal

Build a new pure Go `tgup` repository that preserves the current product behavior with no meaningful user-facing regressions, while the current repository becomes `tgup-python` and is archived after the Go release is proven stable.

This is not a "rewrite for similar functionality". It is a behavior-compatibility migration.

## 2. Non-Negotiable Constraints

1. The new `tgup` repository must be pure Go.
2. The new `tgup` repository must not contain Python runtime requirements, Python packaging, or migration-era Python framing in user-facing product docs.
3. The current repository will be renamed to `tgup-python`.
4. The current repository will be archived only after the Go version passes compatibility, soak, and release gates.
5. The Go version must preserve the existing command surface:
   - `tgup login`
   - `tgup dry-run`
   - `tgup run`
   - `tgup mcp serve`
   - `tgup mcp schema`
6. Existing bug-handling behaviors are part of the required spec and must be preserved, not reinterpreted.
7. State resume semantics must remain stable. The first Go release should prefer direct compatibility with the current `state.sqlite`.
8. Session storage is expected to change. The migration plan assumes users will need to re-login once for the Go version.

## 3. Compatibility Target

The Go implementation is considered compatible when these areas match current behavior closely enough that existing users can switch without changing their workflow:

| Area | Compatibility Requirement |
|---|---|
| CLI | Same subcommands, flags, defaults, and exit code semantics |
| Config | Same precedence: `CLI > ENV > config file > defaults` |
| Scan | Same file discovery, extension filtering, symlink behavior, dedupe behavior |
| Plan | Same ordering, grouping key, album slicing, and preview logic |
| Upload | Same retry classes, duplicate policy, fallback behavior, precheck/postcheck rules |
| State | Same `uploads` identity key and same queue semantics |
| Maintenance | Same trigger conditions and cleanup phases |
| Artifacts | Same artifact names and equivalent content |
| MCP | Same endpoints, auth, SSE resume semantics, and tool contract behavior |

## 4. Source of Truth

The migration spec is defined by all of the following together:

1. Product behavior and defaults in `README.md`
2. Runtime behavior in the current source tree
3. The current automated tests
4. Real-world bug fixes already encoded in upload, queue, metadata, and MCP behavior

Practical anchors in the current repository:

1. `README.md` for command surface, defaults, and user-visible behavior
2. `src/tgup/upload.py` for retry, fallback, metadata, progress, and logging behavior
3. `src/tgup/state.py` for resume keys, queue schema, and maintenance cleanup
4. `src/tgup/mcp/` for protocol and SSE behavior
5. `tests/` for regression protection

## 5. Definition of Done

The migration is complete only when all of the following are true:

1. A new pure Go `tgup` repository ships a production-ready release.
2. The Go CLI passes the agreed compatibility suite.
3. The Go CLI is proven stable in real Telegram usage, including retries, interrupted runs, and queue contention.
4. The old repository has been renamed to `tgup-python`.
5. The old repository publishes final migration guidance and is then archived as read-only.

## 6. Milestones

The recommended execution model is ten milestones. Each milestone has explicit exit criteria and a ready-to-create issue list.

### M0. Freeze the Behavior Contract

Purpose: turn the current implementation into a migration contract before any Go code is relied on.

Deliverables:

1. A compatibility checklist for every command, flag, output, exit code, artifact, and state table.
2. Golden samples for:
   - `--plan` output
   - failure JSON lines
   - `report.json`
   - `report.md`
   - MCP SSE frames
3. Gap analysis for behaviors that are currently implemented but not clearly documented.

Exit Criteria:

1. Every known user-facing behavior is documented or covered by a test.
2. Any "ambiguous but important" behavior is resolved now, not during the port.
3. The Python repo enters feature freeze except migration blockers.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M0-1 | Audit command surface and defaults | Lock all flags, defaults, and exit code semantics |
| M0-2 | Create compatibility checklist | One checklist item per user-visible behavior |
| M0-3 | Capture golden outputs | Save representative output fixtures |
| M0-4 | Identify undocumented behaviors | Promote hidden behavior into explicit contract |
| M0-5 | Freeze migration scope | Mark what is in v1 and what is not |

### M1. Repository Split and Release Strategy

Purpose: set the repository lifecycle and naming plan before public changes.

Deliverables:

1. A new empty Go repository named `tgup`.
2. A rename and archive plan for the current repository to `tgup-python`.
3. A release communication plan covering migration steps and session re-login expectations.

Exit Criteria:

1. Both repositories have clear ownership and purpose.
2. The old repository is not archived early.
3. The new repository starts clean, with no Python product framing.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M1-1 | Create new `tgup` Go repository | Fresh history, fresh project scaffolding |
| M1-2 | Define old repo rename plan | Rename current repo to `tgup-python` later in the process |
| M1-3 | Draft migration release notes | Hosted in old repo only |
| M1-4 | Define archive trigger | Archive only after release and observation window |

### M2. Port Deterministic Core Modules

Purpose: build the non-networked core first, so compatibility work begins on stable logic.

Scope:

1. Config load and validation
2. Path normalization
3. Scan
4. Plan
5. Security helpers
6. Artifact/report generation
7. `dry-run`

Exit Criteria:

1. `dry-run` works end-to-end in Go.
2. The Go implementation matches the current scan and plan behavior.
3. Equivalent tests exist and pass for non-networked logic.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M2-1 | Implement config defaults and merge precedence | Preserve `CLI > ENV > file > defaults` |
| M2-2 | Implement path resolution rules | Relative paths resolve against config file location |
| M2-3 | Implement media scanner | Preserve symlink and extension behavior |
| M2-4 | Implement plan builder | Preserve grouping, ordering, and album max logic |
| M2-5 | Implement dry-run output | Preserve summary and preview structure |
| M2-6 | Implement report writer | Preserve `report.json` and `report.md` semantics |

### M3. Port State, Resume, and Queue Coordination

Purpose: preserve resumability and cross-process correctness.

Scope:

1. `uploads` state table
2. `run_queue` coordination
3. stale runner recovery
4. queue waiting semantics
5. maintenance cleanup
6. `--force-multi-command`

Exit Criteria:

1. The Go version can read the current `state.sqlite`.
2. Resume behavior matches current behavior.
3. FIFO queue semantics are stable under concurrent runs.
4. Cleanup preview/apply behavior is equivalent.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M3-1 | Implement SQLite schema | Preserve table names and key fields |
| M3-2 | Implement resume filtering | Preserve `(path, size, mtime_ns)` identity |
| M3-3 | Implement queue coordinator | Preserve FIFO acquisition and heartbeat |
| M3-4 | Implement stale run recovery | Preserve current liveness behavior |
| M3-5 | Implement maintenance cleanup | Preserve preview, delete, checkpoint, vacuum flow |
| M3-6 | Implement force mode path isolation | Preserve `.force.<pid>.<ts>` style derivation |

### M4. Implement Auth and Session Management with `gotd/td`

Purpose: provide login parity in the new runtime.

Scope:

1. code login
2. 2FA password flow
3. QR login
4. session persistence
5. credential validation and error handling

Exit Criteria:

1. `login --code` works end-to-end.
2. `login --qr` works end-to-end.
3. 2FA flow is stable.
4. Session reuse is reliable.
5. The user-visible behavior remains consistent even though the session format changes.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M4-1 | Design session storage abstraction | Hide `gotd/td` internals behind app-owned interfaces |
| M4-2 | Implement code login | Include retries, prompts, and errors |
| M4-3 | Implement 2FA flow | Preserve prompt and failure behavior |
| M4-4 | Implement QR login | Custom flow if required |
| M4-5 | Implement login error mapping | Normalize auth errors into app-owned error types |
| M4-6 | Document session migration | Old repo only; Go repo stays product-focused |

### M5. Port the Upload Engine

Purpose: reproduce the most bug-sensitive part of the product.

This milestone is the highest-risk and highest-value milestone.

Scope:

1. target resolution
2. single-file upload
3. album upload
4. concurrency control
5. retry policy
6. duplicate handling
7. document fallback paths
8. cancellation
9. progress output
10. state writes
11. artifact logging

Exit Criteria:

1. `run` works with real Telegram uploads.
2. Retry and fallback behavior match the current app closely.
3. Partial success and failure handling preserve state correctly.
4. Progress output and logs remain operationally useful and stable.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M5-1 | Implement app-owned Telegram client wrapper | No `gotd/td` types outside transport layer |
| M5-2 | Implement single media send path | Preserve success and fallback rules |
| M5-3 | Implement album send path | Preserve caption and slicing semantics |
| M5-4 | Implement retry classifier | Preserve flood-wait vs backoff behavior |
| M5-5 | Implement duplicate policy | Preserve `skip`, `ask`, and `upload` behavior |
| M5-6 | Implement cancellation path | Preserve interrupt semantics and final state |
| M5-7 | Implement progress renderer | Preserve dual-layer progress behavior |
| M5-8 | Implement artifact log sink | Preserve upload log event usefulness |
| M5-9 | Run real Telegram integration passes | Required before promotion |

### M6. Port Metadata Validation and Thumbnail Generation

Purpose: preserve the video-specific protections that prevent invalid Telegram rendering.

Scope:

1. metadata probe providers
2. `ffprobe` fallback
3. strict metadata policy
4. post-upload video attribute validation
5. thumbnail generation

Exit Criteria:

1. Pre-upload validation catches the same invalid media classes.
2. Post-upload validation still detects bad Telegram-side video metadata.
3. Thumbnail generation behavior matches current expectations.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M6-1 | Implement metadata probe abstraction | Keep provider choice isolated |
| M6-2 | Implement `ffprobe` fallback | Preserve warning behavior |
| M6-3 | Implement strict metadata enforcement | Preserve block vs warn behavior |
| M6-4 | Implement post-upload validation | Preserve `duration`, dimensions, streaming checks |
| M6-5 | Implement thumbnail generation pipeline | Preserve `auto` and `off` behavior |
| M6-6 | Build real media regression corpus | Required for confidence |

### M7. Port MCP Service

Purpose: preserve automation and external control capabilities.

Scope:

1. HTTP endpoints
2. bearer token auth
3. origin checks
4. allow-root path enforcement
5. job management
6. event store
7. SSE replay and resume
8. schema export

Exit Criteria:

1. The Go MCP service matches the current protocol behavior.
2. SSE replay, resume, keepalive, and retention behavior are stable.
3. Security behavior matches current constraints.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M7-1 | Implement MCP config and validation | Preserve defaults and required fields |
| M7-2 | Implement HTTP handlers | Preserve route and status code behavior |
| M7-3 | Implement auth and path validation | Preserve token, origin, and root checks |
| M7-4 | Implement event store | Preserve replay and cleanup behavior |
| M7-5 | Implement SSE transport | Preserve `Last-Event-ID` semantics |
| M7-6 | Implement job manager | Preserve run start, status, cancel behavior |
| M7-7 | Implement schema export | Preserve contract utility |

### M8. Compatibility, Shadow Runs, and Soak Testing

Purpose: validate the Go port against the real product contract.

Scope:

1. side-by-side comparisons
2. fixture-based comparisons
3. stress and interruption tests
4. queue contention tests
5. long-running soak tests

Exit Criteria:

1. No unresolved P0 or P1 regressions remain.
2. Any accepted behavior differences are documented and explicit.
3. The Go release is operationally trusted.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M8-1 | Build side-by-side comparison harness | Compare output, artifacts, and state |
| M8-2 | Run upload shadow tests | Same inputs, separate sessions |
| M8-3 | Run queue contention tests | Multi-process coordination |
| M8-4 | Run interruption and recovery tests | SIGINT, SIGTERM, network loss |
| M8-5 | Run long-duration soak tests | Required before public cutover |
| M8-6 | Log and classify remaining deltas | Accept or fix, but do not ignore |

### M9. Release Cutover and Old Repo Archive

Purpose: switch the public product identity cleanly.

Scope:

1. Release the Go version
2. rename current repo to `tgup-python`
3. publish migration instructions in the old repo
4. observe production usage
5. archive old repo

Exit Criteria:

1. The Go release is public and stable.
2. The old repo clearly points users to the new repo.
3. The old repo is archived only after a successful observation window.

Suggested Issues:

| ID | Title | Notes |
|---|---|---|
| M9-1 | Cut Go `tgup` release | First supported production release |
| M9-2 | Rename old repo to `tgup-python` | Preserve discoverability |
| M9-3 | Publish migration guide in old repo | Include re-login expectations |
| M9-4 | Monitor adoption window | Capture real upgrade issues |
| M9-5 | Archive old repo | Final read-only state |

## 7. High-Risk Areas

These areas deserve explicit owner attention because they are the most likely to create false confidence during a rewrite.

1. QR login parity with `gotd/td`
2. Mapping Telegram and transport errors into the app's existing retry model
3. Album send semantics and caption behavior
4. Video metadata precheck and postcheck behavior
5. State compatibility with existing `state.sqlite`
6. SSE replay and `Last-Event-ID` resume semantics
7. Cross-process queue behavior under concurrent runs

## 8. Release Gating Checklist

Do not ship or archive based on "it mostly works". Ship only when these gates are green:

1. Deterministic modules pass the compatibility suite.
2. State and queue logic pass concurrency and recovery tests.
3. Real Telegram login and upload flows are proven stable.
4. MCP service passes protocol and resume tests.
5. Side-by-side output and artifact comparison is acceptable.
6. Soak tests complete without blocking regressions.
7. Migration notes are ready in the old repo.
8. Rollback plan is defined before public cutover.

## 9. Suggested Execution Order

The recommended order is:

1. M0
2. M1
3. M2
4. M3
5. M4
6. M5
7. M6
8. M7
9. M8
10. M9

The most important sequencing rule is this: do not archive the current repository until M8 is complete and the Go release has cleared the observation window.

