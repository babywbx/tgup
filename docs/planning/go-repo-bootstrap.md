# TGUP Go Repository Bootstrap Spec

Status: Draft  
Target: New pure Go repository named `tgup`  
Transport: `gotd/td`

## 1. Product Rules for the New Repository

The new repository is a fresh Go product repository, not a migration narrative repository.

Rules:

1. No Python packaging files.
2. No Python runtime requirements.
3. No user-facing language like "ported from Python" or "legacy Python version" in the new repo README.
4. No direct leakage of `gotd/td` types outside the Telegram transport layer.
5. Behavior compatibility is enforced by tests and fixtures, not by ad hoc manual checks.

Historical migration notes belong in the old `tgup-python` repository only.

## 2. Initial Repository Layout

Recommended initial tree:

```text
tgup/
  .github/
    workflows/
      ci.yml
      release.yml
  cmd/
    tgup/
      main.go
  internal/
    app/
      run.go
      login.go
      dryrun.go
      mcp.go
    artifacts/
      writer.go
      report.go
    cli/
      root.go
      login.go
      dryrun.go
      run.go
      mcp.go
    config/
      defaults.go
      loader.go
      merge.go
      validate.go
      env.go
      file.go
      types.go
    files/
      paths.go
      fs.go
    logging/
      logger.go
      redact.go
    media/
      metadata.go
      ffprobe.go
      thumbnail.go
      kinds.go
    mcp/
      config.go
      http.go
      handlers.go
      jobs.go
      events.go
      schemas.go
      security.go
    plan/
      build.go
      model.go
      sort.go
    progress/
      render.go
      terminal.go
    queue/
      coordinator.go
      heartbeat.go
    scan/
      scan.go
      filter.go
      model.go
    state/
      schema.go
      store.go
      cleanup.go
      models.go
    tg/
      auth.go
      client.go
      errors.go
      session.go
      uploader.go
      types.go
    upload/
      run.go
      retry.go
      precheck.go
      postcheck.go
      duplicate.go
      summary.go
    xerrors/
      codes.go
      wrap.go
  testdata/
    golden/
      dryrun/
      reports/
      mcp/
    media/
      valid/
      invalid/
  docs/
    architecture.md
    compatibility-matrix.md
  go.mod
  go.sum
  LICENSE
  README.md
```

Notes:

1. Keep all app code under `internal/` until there is a real reason to publish reusable packages.
2. The package split is functional, not layered by framework.
3. `cmd/tgup` should stay small and only wire the app together.

## 3. Module Path and Versioning

Recommended module path:

```go
module github.com/<owner>/tgup
```

Rules:

1. Start at `v0` while behavior compatibility is still being proven.
2. Do not mark `v1` until the Go version has replaced the old primary distribution.
3. Keep a clean semantic version history from day one.

## 4. Package Responsibilities

| Package | Responsibility | Key Constraint |
|---|---|---|
| `internal/cli` | Parse subcommands, flags, and usage text | Must preserve current command surface |
| `internal/app` | Application-level orchestration | No transport-specific code |
| `internal/config` | Defaults, file/env/CLI merge, validation | Preserve precedence and path rules |
| `internal/scan` | Discover candidate files | Preserve filtering and symlink semantics |
| `internal/plan` | Grouping, sorting, album slicing | Preserve current planning behavior |
| `internal/state` | SQLite persistence | Preserve resume keys and cleanup flow |
| `internal/queue` | Cross-process run coordination | Preserve FIFO and heartbeat behavior |
| `internal/tg` | `gotd/td` adapter layer | The only place allowed to know `gotd/td` |
| `internal/upload` | Upload business logic | Must stay independent from `gotd/td` types |
| `internal/media` | Metadata and thumbnail helpers | External tools isolated behind interfaces |
| `internal/progress` | Terminal progress rendering | Stable operator-facing output |
| `internal/artifacts` | `upload.log`, `report.json`, `report.md` | Preserve artifact contract |
| `internal/mcp` | HTTP control plane, SSE, schemas | Preserve protocol semantics |
| `internal/logging` | Output and redaction helpers | Preserve sensitive value handling |
| `internal/xerrors` | App-owned error taxonomy | No raw transport errors crossing package boundaries |

## 5. Core Data Model

These types should be owned by the application and reused across packages:

1. `scan.Item`
2. `plan.Album`
3. `plan.Plan`
4. `state.UploadRow`
5. `state.CleanupReport`
6. `upload.Summary`
7. `artifacts.RunArtifacts`
8. `mcp.Job`
9. `mcp.Event`

Important rule: do not let transport-specific types become part of the core data model.

## 6. First-Batch Interfaces

The goal of the initial interfaces is to isolate unstable dependencies early.

### 6.1 Clock and Randomness

```go
package app

import "time"

type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type RNG interface {
	Float64() float64
}
```

Reason:

1. Retry backoff
2. Queue heartbeat timing
3. Test determinism

### 6.2 Config Loading

```go
package config

type Loader interface {
	Load(path string) (LoadedConfig, error)
}

type Validator interface {
	Validate(ResolvedConfig) error
}
```

`LoadedConfig` should keep both parsed values and source metadata such as config file directory for path resolution.

### 6.3 Filesystem Abstraction

```go
package files

import "io/fs"

type FS interface {
	Stat(name string) (fs.FileInfo, error)
	Lstat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Open(name string) (fs.File, error)
}
```

Reason:

1. Scan logic becomes testable
2. Symlink behavior is easier to verify
3. Some path safety checks become deterministic in tests

### 6.4 State Store

```go
package state

import "context"

type Store interface {
	MarkSent(ctx context.Context, in MarkSentInput) error
	MarkFailed(ctx context.Context, in MarkFailedInput) error
	IsDone(ctx context.Context, item ResumeKey) (bool, error)
	ListPending(ctx context.Context, items []ResumeKey) ([]ResumeKey, error)
	GetUploadRow(ctx context.Context, item ResumeKey) (*UploadRow, error)
	ApplyMaintenance(ctx context.Context, cfg MaintenanceConfig, force bool) (CleanupReport, error)
	Close() error
}
```

`ResumeKey` must preserve the current identity semantics:

1. absolute resolved path
2. file size
3. `mtime_ns`

### 6.5 Queue Coordination

```go
package queue

import "context"

type Coordinator interface {
	RunID() string
	WaitUntilTurn(ctx context.Context, onWait func(ahead int)) error
	Heartbeat(ctx context.Context) error
	Finish(ctx context.Context, status string) error
	Cancel(ctx context.Context) error
	Close() error
}
```

This package owns cross-process coordination and should not depend on upload internals.

### 6.6 Telegram Auth

```go
package tg

import "context"

type AuthService interface {
	SendCode(ctx context.Context, phone string) error
	SignInWithCode(ctx context.Context, phone, code string) error
	SignInWithPassword(ctx context.Context, password string) error
	StartQRLogin(ctx context.Context) (QRLogin, error)
	WaitQRLogin(ctx context.Context, qr QRLogin) error
}

type QRLogin struct {
	URL string
}
```

Notes:

1. Keep QR as an app-owned type.
2. The CLI should not know whether the implementation uses helpers or a custom `gotd/td` flow.

### 6.7 Telegram Upload Transport

```go
package tg

import "context"

type Transport interface {
	ResolveTarget(ctx context.Context, target string) (ResolvedTarget, error)
	SendSingle(ctx context.Context, req SendSingleRequest) (SendResult, error)
	SendAlbum(ctx context.Context, req SendAlbumRequest) (SendResult, error)
}

type ProgressFunc func(sentBytes int64, totalBytes int64)

type SendSingleRequest struct {
	Target            ResolvedTarget
	Path              string
	Caption           string
	ParseMode         string
	ForceDocument     bool
	SupportsStreaming bool
	ThumbnailPath     string
	Progress          ProgressFunc
}

type SendAlbumRequest struct {
	Target            ResolvedTarget
	Items             []AlbumMedia
	ParseMode         string
	Progress          ProgressFunc
}

type AlbumMedia struct {
	Path              string
	Caption           string
	ForceDocument     bool
	SupportsStreaming bool
	ThumbnailPath     string
}

type SendResult struct {
	MessageIDs []int
	GroupID    string
	Messages   []SentMessage
}
```

Important rule: `SentMessage` is an app-owned projection of the returned Telegram message. It should contain only the fields needed for post-upload validation and state bookkeeping.

### 6.8 Metadata Probe

```go
package media

import "context"

type MetadataProber interface {
	ProbeVideo(ctx context.Context, path string) (VideoMetadata, error)
}

type VideoMetadata struct {
	DurationSeconds float64
	Width           int
	Height          int
}
```

This lets the app preserve current validation behavior while swapping implementations later.

### 6.9 Thumbnail Generation

```go
package media

import "context"

type Thumbnailer interface {
	ExtractVideoThumbnail(ctx context.Context, videoPath string) (thumbnailPath string, cleanup func(), err error)
}
```

The returned cleanup callback prevents temp file leaks and makes ownership explicit.

### 6.10 Artifact Writing

```go
package artifacts

type LogSink interface {
	Write(level string, message string) error
	Close() error
}

type ReportWriter interface {
	WriteJSON(summary any) error
	WriteMarkdown(summary any) error
}
```

### 6.11 MCP Event Store

```go
package mcp

import "context"

type EventStore interface {
	Append(ctx context.Context, event Event) (Event, error)
	List(ctx context.Context, jobID string, sinceSeq int64, limit int) (events []Event, hasMore bool, err error)
	Register(jobID string) Subscription
	Cleanup(ctx context.Context) error
}

type Subscription interface {
	C() <-chan Event
	Close()
}
```

## 7. Error Taxonomy

Define app-owned errors early. Do not let raw transport or library errors leak into the app boundary.

Minimum error categories:

1. `ConfigError`
2. `ScanError`
3. `UploadError`
4. `AuthError`
5. `StateError`
6. `SecurityError`
7. `MCPError`
8. `RetryableError`
9. `NonRetryableError`
10. `InterruptedError`

Rules:

1. Retry classification happens once, close to the transport boundary.
2. CLI exit codes map from app-owned error categories.
3. Logs may include normalized reason codes but not raw secrets.

## 8. Logging and Output Rules

The Go version should preserve operational output discipline from the current product.

Rules:

1. Keep tagged logs such as queue, maintenance, mcp, and done-style status messages.
2. Preserve machine-readable failure JSON lines for upload failures.
3. Keep redaction centralized.
4. Progress rendering must tolerate non-interactive terminals and log sinks.
5. Any output intended for automation should be explicitly versioned or kept stable.

## 9. State Compatibility Rules

The first Go release should target these compatibility rules:

1. Same `uploads` primary key semantics.
2. Same meaning of `status='sent'` and `status='failed'`.
3. Same `run_queue` lifecycle semantics.
4. Same maintenance trigger logic.
5. Same best-effort read compatibility with existing `state.sqlite`.

Session compatibility is intentionally excluded. The new repo should use its own session format and require a one-time re-login.

## 10. Testing Strategy

The repository should be test-first for every behavior that used to be protected in the Python codebase.

Test layers:

1. Unit tests for config, scan, plan, security, retry, and cleanup logic
2. State tests against a real temporary SQLite database
3. Integration tests using fake Telegram transport implementations
4. MCP protocol tests against a live in-process HTTP server
5. Golden tests for:
   - dry-run output
   - report files
   - failure JSON lines
   - SSE event frames
6. Real Telegram opt-in integration tests gated behind explicit environment variables

Rules:

1. A fake transport must exist before the real transport is fully implemented.
2. No business logic tests should require real Telegram.
3. Real Telegram tests are required before release, but not for every local test run.

## 11. CI and Release Baseline

Initial CI should include:

1. `go test ./...`
2. `go test -race ./...`
3. `go vet ./...`
4. formatting check
5. lint check
6. cross-platform build smoke checks for macOS, Linux, and Windows

Release pipeline baseline:

1. Build reproducible binaries
2. Attach checksums
3. Run a release-only integration gate
4. Publish release notes

## 12. First Implementation Sequence

This is the recommended order to start writing code in the new Go repository:

1. Create repo skeleton, `go.mod`, CI, and package placeholders.
2. Implement app-owned models and error taxonomy.
3. Implement config loading and validation.
4. Implement scan and plan logic.
5. Implement SQLite state store and queue coordinator.
6. Implement artifact writers and dry-run.
7. Implement fake `tg.Transport` and wire upload orchestration against the fake.
8. Implement real `tg` auth and upload transport with `gotd/td`.
9. Implement metadata and thumbnail helpers.
10. Implement MCP service.
11. Backfill golden fixtures and side-by-side compatibility tests.

This sequence keeps the bulk of the business logic independent from Telegram transport details for as long as possible.

## 13. Immediate Start Checklist

The project is ready to begin once these concrete setup steps are done:

1. Create the new `tgup` Go repository.
2. Copy this bootstrap spec into the new repo's planning docs.
3. Create the package skeleton exactly once.
4. Open the first milestone issues from the migration roadmap.
5. Start with config, scan, and plan before auth or upload.

