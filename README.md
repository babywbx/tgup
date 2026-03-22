<div align="center"><a name="readme-top"></a>

# tgup

High-throughput Telegram Saved Messages media uploader CLI.<br/>
Built on gotd (MTProto) for batch photo/video uploads with resume, queue coordination, and parallel isolation.

**English** · [简体中文](./README.zh-CN.md) · [Changelog][github-release-link] · [Issues][github-issues-link]

<!-- SHIELD GROUP -->

[![][github-release-shield]][github-release-link]
[![][github-releasedate-shield]][github-releasedate-link]
[![][github-action-ci-shield]][github-action-ci-link]
[![][github-license-shield]][github-license-link]<br/>
[![][github-contributors-shield]][github-contributors-link]
[![][github-forks-shield]][github-forks-link]
[![][github-stars-shield]][github-stars-link]
[![][github-issues-shield]][github-issues-link]<br/>
[![][go-version-shield]][go-version-link]
[![][go-report-shield]][go-report-link]

</div>

<details>
<summary><kbd>Table of contents</kbd></summary>

#### TOC

- [✨ Features](#-features)
- [📋 Telegram Constraints](#-telegram-constraints)
- [📦 Installation](#-installation)
- [🚀 Quick Start](#-quick-start)
- [⌨️ Commands](#️-commands)
- [🔀 Run Modes](#-run-modes)
- [⚙️ Configuration](#️-configuration)
- [🌐 Environment Variables](#-environment-variables)
- [🔍 Key Behaviors](#-key-behaviors)
- [📁 Project Structure](#-project-structure)
- [❓ FAQ](#-faq)
- [🛠 Development](#-development)

####

<br/>

</details>

## ✨ Features

> \[!IMPORTANT]
>
> **Star Us** — you will receive all release notifications from GitHub without any delay \~ ⭐️

`tgup` solves one problem well: **reliably upload local media to Telegram Saved Messages**. Core goals:

- **🎯 Predictable** — scan, group, sort, upload with clear plans
- **🔄 Resumable** — SQLite state DB with checkpoint resume
- **🤝 Coordinated** — default FIFO queue prevents multi-terminal conflicts
- **⚡ Parallel** — force-parallel mode with isolated state/session when needed

<table>
<tr><th>Feature</th><th>Default</th><th>Flags</th><th>Notes</th></tr>
<tr><td>🔐 Login (Code / QR)</td><td>Reuse session</td><td><code>login --code</code> / <code>login --qr</code></td><td>2FA supported</td></tr>
<tr><td>🧪 Quick verify</td><td>Generate test images → upload → cleanup</td><td><code>demo</code></td><td>One-command E2E verification</td></tr>
<tr><td>📂 Media scan</td><td>Recursive, no symlinks</td><td><code>--src</code> <code>--recursive</code> <code>--include-ext</code></td><td>Multiple <code>--src</code>, dedup by real path</td></tr>
<tr><td>📑 Group & slice</td><td>Group by parent dir, slice by 10</td><td><code>--order</code> <code>--album-max</code></td><td>Telegram album limit = 10</td></tr>
<tr><td>🚀 Album concurrency</td><td>5 albums in parallel</td><td><code>--concurrency-album</code></td><td>Concurrency unit = album</td></tr>
<tr><td>🧩 Chunked upload</td><td>8 threads per file</td><td><code>--threads</code></td><td>512 KB chunks + multi-threaded</td></tr>
<tr><td>🔌 DC connection pool</td><td>8 MTProto connections</td><td><code>--pool-size</code></td><td>Break single-connection bandwidth limit</td></tr>
<tr><td>💾 Resume</td><td>Enabled</td><td><code>--resume</code> / <code>--no-resume</code></td><td>Identify by <code>path + size + mtime_ns</code></td></tr>
<tr><td>📋 Duplicate policy</td><td><code>ask</code></td><td><code>--duplicate {skip,ask,upload}</code></td><td>Only effective with resume on</td></tr>
<tr><td>🔒 Queue coordination</td><td>Same-state FIFO queue</td><td>Default</td><td>Cross-process SQLite <code>run_queue</code></td></tr>
<tr><td>⚡ Force parallel</td><td>Off</td><td><code>--force-multi-command</code></td><td>Auto-isolate state/session</td></tr>
<tr><td>🧹 State maintenance</td><td>Enabled</td><td><code>--maintenance</code> <code>--cleanup-now</code></td><td>Time / size / row-count triggers</td></tr>
<tr><td>📊 Progress bar</td><td>On</td><td><code>--no-progress</code></td><td>Total bytes + album/files status</td></tr>
<tr><td>📝 Upload plan preview</td><td>Off</td><td><code>--plan</code> <code>--plan-files</code></td><td>Preview grouping before upload</td></tr>
<tr><td>🌐 MCP HTTP server</td><td>Off</td><td><code>mcp serve</code></td><td>Streamable HTTP + SSE</td></tr>
</table>

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📋 Telegram Constraints

> \[!NOTE]
>
> These are Telegram platform limitations, not tgup limitations.

- Max **10** media per album (hard limit)
- Only **1** caption per album (assigned to first media)
- MTProto client login only — **not** Bot API

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📦 Installation

- **Go** >= 1.25
- Single binary, no external runtime dependencies

Download pre-built binaries from [GitHub Releases][github-release-link] (Linux / macOS / Windows, amd64 / arm64), or build from source:

```bash
go build -o tgup ./cmd/tgup
```

> \[!TIP]
>
> - Video metadata requires system `ffprobe` (part of FFmpeg). Without it, Telegram may show `duration=0 / 1x1` for videos.
> - tgup pre-checks `ffprobe` availability and warns if missing.
> - SQLite uses pure Go (`modernc.org/sqlite`) — no CGo required.

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🚀 Quick Start

**1.** Get `api_id` and `api_hash` from [my.telegram.org](https://my.telegram.org)

**2.** Login and create a session:

```bash
tgup login --code
# or
tgup login --qr
```

**3.** *(Optional)* Quick E2E verification:

```bash
tgup demo
```

**4.** Preview the upload plan:

```bash
tgup dry-run --src /path/to/media --order mtime
```

**5.** Start uploading:

```bash
tgup run --src /path/to/media --caption "daily"
```

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ⌨️ Commands

### Top-level

```bash
tgup [-h] {login,dry-run,run,demo,mcp,version}
```

### `demo`

```bash
tgup demo [--config CONFIG] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION]
```

Quick E2E verification: generate 2 test images → upload to Saved Messages → auto cleanup.

### `login`

```bash
tgup login [--config CONFIG] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION] (--qr | --code) [--phone PHONE]
```

| Flag | Description |
|------|-------------|
| `--qr` | QR code login in terminal |
| `--code` | SMS code login, prompts for 2FA if needed |
| `--session` | Session file path (default `./secrets/session.session`) |

### `dry-run`

```bash
tgup dry-run [--src SRC] [--recursive] [--follow-symlinks] [--include-ext CSV] [--exclude-ext CSV] [--order {name,mtime,size,random}] [--reverse] [--album-max N]
```

Scan + build plan only, no upload. Prints file counts, image/video breakdown, album list.

### `run`

```bash
tgup run [--src SRC] [--caption CAPTION] [--concurrency-album N] [--threads N] [--pool-size N] [--duplicate {skip,ask,upload}] [--force-multi-command] [--plan] ...
```

Key flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | `me` | Upload target |
| `--parse-mode` | `plain` | `plain` or `md` |
| `--concurrency-album` | `5` | Album-level concurrency |
| `--threads` | `8` | Per-file parallel chunks (512 KB each) |
| `--pool-size` | `8` | DC connection pool size (`0` to disable) |
| `--strict-metadata` | off | Reject album on bad video metadata |
| `--image-mode` | `auto` | Image send strategy |
| `--video-thumbnail` | `auto` | Video thumbnail strategy |
| `--state` | `./data/state.sqlite` | SQLite state DB path |
| `--artifacts-dir` | `./data/runs` | Per-run artifacts root |
| `--duplicate` | `ask` | `ask` / `skip` / `upload` for sent files |
| `--plan` | off | Print plan before upload |

### `mcp`

```bash
tgup mcp serve [--host HOST] [--port PORT] [--token TOKEN] [--allow-root PATH ...] [--enable-sse]
tgup mcp schema --out /path/to/schema.json
```

- `mcp serve` — local MCP Streamable HTTP server, default `127.0.0.1:8765`
- `mcp schema` — export all MCP tool JSON Schemas
- All `/mcp` requests require `Bearer token`
- `GET /mcp` + `Accept: text/event-stream` — subscribe to SSE events (supports `Last-Event-ID`)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🔀 Run Modes

### Default Queue Mode (Recommended)

Multiple terminals running `tgup run` with the same `--state`:

- Auto-enter unified FIFO queue
- Only the head-of-queue task uploads
- Others wait with `waiting ahead=N`

Best for: same media library + shared checkpoint state.

### Force Parallel Mode

```bash
tgup run --src /path/to/media --force-multi-command
```

- Bypass global queue
- Auto-derive isolated state/session (`.force.<pid>.<ts>` suffix)
- No shared checkpoint state
- Auto-disable maintenance (skip temp state cleanup)

Best for: independent batch jobs running in parallel.

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ⚙️ Configuration

**Priority:** `CLI > ENV > config file > defaults`

### Config File Sources

| Source | Path |
|--------|------|
| Global | `~/.config/tgup/config.toml` |
| Project | `./tgup.toml` (overrides global) |
| Explicit | `--config /path/to/config.toml` (exclusive) |

> \[!NOTE]
>
> Relative paths in config files resolve relative to the **config file's directory**, not the current shell directory.
> Affected fields: `telegram.session`, `paths.state`, `scan.src`, `mcp.allow_roots`, `mcp.control_db`.

Full config template: [`tgup.example.toml`](tgup.example.toml)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🌐 Environment Variables

<details>
<summary><kbd>Core</kbd></summary>

| Variable | Description |
|----------|-------------|
| `TGUP_API_ID` | Telegram API ID |
| `TGUP_API_HASH` | Telegram API Hash |
| `TGUP_SESSION` / `TGUP_SESSION_PATH` | Session path (mutually exclusive) |
| `TGUP_STATE` / `TGUP_STATE_PATH` | State DB path (mutually exclusive) |
| `TGUP_ARTIFACTS_DIR` | Artifacts root directory |
| `TGUP_THREADS` | Per-file upload threads |
| `TGUP_POOL_SIZE` | DC connection pool size |

</details>

<details>
<summary><kbd>Maintenance</kbd></summary>

| Variable | Description |
|----------|-------------|
| `TGUP_MAINTENANCE_ENABLED` | Enable maintenance |
| `TGUP_MAINTENANCE_INTERVAL_HOURS` | Cleanup interval |
| `TGUP_MAINTENANCE_RETENTION_SENT_DAYS` | Sent record retention |
| `TGUP_MAINTENANCE_RETENTION_FAILED_DAYS` | Failed record retention |
| `TGUP_MAINTENANCE_RETENTION_QUEUE_DAYS` | Queue record retention |
| `TGUP_MAINTENANCE_MAX_DB_MB` | Max DB size trigger |
| `TGUP_MAINTENANCE_MAX_UPLOAD_ROWS` | Max row count trigger |
| `TGUP_MAINTENANCE_FIRST_RUN_PREVIEW` | Preview before first cleanup |
| `TGUP_MAINTENANCE_VACUUM_COOLDOWN_HOURS` | VACUUM cooldown |
| `TGUP_MAINTENANCE_VACUUM_MIN_RECLAIM_MB` | Min reclaimable for VACUUM |

</details>

<details>
<summary><kbd>MCP</kbd></summary>

| Variable | Description |
|----------|-------------|
| `TGUP_MCP_ENABLED` | Enable MCP server |
| `TGUP_MCP_HOST` | Listen host |
| `TGUP_MCP_PORT` | Listen port |
| `TGUP_MCP_TOKEN` | Bearer token |
| `TGUP_MCP_ALLOW_ROOTS` | Allowed root paths |
| `TGUP_MCP_CONTROL_DB` | Control DB path |
| `TGUP_MCP_EVENT_RETENTION_HOURS` | Event retention |
| `TGUP_MCP_MAX_CONCURRENT_JOBS` | Max concurrent jobs |
| `TGUP_MCP_ENABLE_SSE` | Enable SSE |
| `TGUP_MCP_ALLOWED_ORIGINS` | CORS origins |

</details>

> Boolean values: `1/0`, `true/false`, `yes/no`, `on/off`

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🔍 Key Behaviors

### Scan & Grouping

- Group by `src_root + parent_dir` — different `--src` roots never mix
- Default media extensions:
  - Image: `.jpg` `.jpeg` `.png` `.webp` `.heic`
  - Video: `.mp4` `.mov` `.mkv` `.webm`

### Upload & Retry

- Album and single-file uploads supported
- `FloodWait` — wait per Telegram's cooldown, then retry
- Other errors — exponential backoff with jitter
- `ImageProcessFailedError` — auto-fallback to document
- Video metadata pre-check: `duration > 0 && width > 1 && height > 1`
- Post-upload verification: `DocumentAttributeVideo` with `supports_streaming = true`
- Per-run artifacts: `data/runs/<run_id>/upload.log`, `report.json`, `report.md`

### Resume

- `uploads` table key: `(path, size, mtime_ns)`
- `status='sent'` = completed, skipped on next run
- `--duplicate` only controls re-upload of already-sent items

### Maintenance

Triggers (any condition met):

- Time since last cleanup > `interval_hours`
- DB size > `max_db_mb`
- Upload rows > `max_upload_rows`

Cleanup flow: preview on first run → delete expired rows → evaluate reclaimable space → `VACUUM` if threshold met.

Use `--cleanup-now` to skip preview and execute immediately.

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📁 Project Structure

```
cmd/tgup/           Entry point
internal/
  app/               Command orchestration & business logic
  artifacts/          Run artifacts (logs, reports)
  cli/                Argument parsing & command entry
  config/             TOML config loading, merging, validation
  files/              Filesystem abstraction, path safety
  logging/            Structured logging & sanitization
  mcp/                MCP HTTP control plane & SSE
  media/              Video metadata & thumbnail extraction
  plan/               Album grouping & sorting
  progress/           Terminal upload progress rendering
  queue/              Cross-process FIFO queue coordination
  scan/               File discovery & extension filtering
  state/              SQLite state persistence & maintenance
  tg/                 Telegram transport layer (gotd adapter)
  upload/             Upload orchestration, retry & failure handling
  xerrors/             Application-level error classification
```

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ❓ FAQ

<details>
<summary><kbd>Q: Error <code>missing api_id</code> / <code>missing api_hash</code></kbd></summary>

Credentials not found in the config chain. Check:

1. Did you pass `--api-id` / `--api-hash`?
2. Are `TGUP_API_ID` / `TGUP_API_HASH` set?
3. Does `tgup.toml` contain a `[telegram]` section?

</details>

<details>
<summary><kbd>Q: Why is only one terminal uploading?</kbd></summary>

This is the default FIFO coordination — prevents concurrent state conflicts.
Use `--force-multi-command` for true parallel execution.

</details>

<details>
<summary><kbd>Q: Why does the first cleanup only show a preview?</kbd></summary>

Default `first_run_preview=true` — shows "what would be deleted" first.
Use `--cleanup-now` or set `first_run_preview=false` to execute immediately.

</details>

<details>
<summary><kbd>Q: Why are some images sent as documents?</kbd></summary>

Telegram's `ImageProcessFailedError` triggers auto-fallback to document upload. This is intentional fault tolerance.

</details>

<details>
<summary><kbd>Q: Why do videos show as 0s / 1×1 in Telegram?</kbd></summary>

Video metadata extraction failed. Check:

1. Is `ffprobe` installed? (`ffprobe -version`)
2. Any `precheck` / `ffprobe_missing` warnings in upload logs?
3. Enable `--strict-metadata` to block bad-metadata uploads

</details>

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🛠 Development

```bash
# Build
go build ./cmd/tgup

# Test (with race detection)
go test -count=1 -race ./...

# Format check
gofmt -l .
go vet ./...
```

> \[!TIP]
>
> See also: [`AI.md`](AI.md) · [`MCP.md`](MCP.md)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

---

<details><summary><h4>📝 License</h4></summary>

This project is licensed under the [Apache License 2.0](./LICENSE).

</details>

Copyright © 2025 [babywbx][profile-link]. <br />
This project is [Apache-2.0][github-license-link] licensed.

<!-- LINK GROUP -->

[back-to-top]: https://img.shields.io/badge/-BACK_TO_TOP-151515?style=flat-square
[github-action-ci-link]: https://github.com/babywbx/tgup/actions/workflows/ci.yml
[github-action-ci-shield]: https://img.shields.io/github/actions/workflow/status/babywbx/tgup/ci.yml?label=CI&labelColor=black&logo=githubactions&logoColor=white&style=flat-square
[github-contributors-link]: https://github.com/babywbx/tgup/graphs/contributors
[github-contributors-shield]: https://img.shields.io/github/contributors/babywbx/tgup?color=c4f042&labelColor=black&style=flat-square
[github-forks-link]: https://github.com/babywbx/tgup/network/members
[github-forks-shield]: https://img.shields.io/github/forks/babywbx/tgup?color=8ae8ff&labelColor=black&style=flat-square
[github-issues-link]: https://github.com/babywbx/tgup/issues
[github-issues-shield]: https://img.shields.io/github/issues/babywbx/tgup?color=ff80eb&labelColor=black&style=flat-square
[github-license-link]: https://github.com/babywbx/tgup/blob/main/LICENSE
[github-license-shield]: https://img.shields.io/badge/license-Apache%202.0-white?labelColor=black&style=flat-square
[github-release-link]: https://github.com/babywbx/tgup/releases
[github-release-shield]: https://img.shields.io/github/v/release/babywbx/tgup?color=369eff&labelColor=black&logo=github&style=flat-square
[github-releasedate-link]: https://github.com/babywbx/tgup/releases
[github-releasedate-shield]: https://img.shields.io/github/release-date/babywbx/tgup?labelColor=black&style=flat-square
[github-stars-link]: https://github.com/babywbx/tgup/stargazers
[github-stars-shield]: https://img.shields.io/github/stars/babywbx/tgup?color=ffcb47&labelColor=black&style=flat-square
[go-report-link]: https://goreportcard.com/report/github.com/babywbx/tgup
[go-report-shield]: https://img.shields.io/badge/go%20report-A+-brightgreen?labelColor=black&style=flat-square
[go-version-link]: https://github.com/babywbx/tgup/blob/main/go.mod
[go-version-shield]: https://img.shields.io/badge/go-%3E%3D%201.25-00ADD8?labelColor=black&logo=go&logoColor=white&style=flat-square
[profile-link]: https://github.com/babywbx
