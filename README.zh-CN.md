<div align="center"><a name="readme-top"></a>

# tgup

高吞吐 Telegram 收藏夹（Saved Messages）媒体上传 CLI。<br/>
基于 gotd（MTProto），面向图片/视频批量上传，支持断点续传、队列协调和多命令并行隔离。

[English](./README.md) · **简体中文** · [更新日志][github-release-link] · [反馈][github-issues-link]

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
<summary><kbd>目录</kbd></summary>

#### TOC

- [✨ 功能特性](#-功能特性)
- [📋 Telegram 约束](#-telegram-约束)
- [📦 安装](#-安装)
- [🚀 快速开始](#-快速开始)
- [⌨️ 命令说明](#️-命令说明)
- [🔀 运行模式](#-运行模式)
- [⚙️ 配置系统](#️-配置系统)
- [🌐 环境变量](#-环境变量)
- [🔍 关键行为](#-关键行为)
- [📁 项目结构](#-项目结构)
- [❓ 常见问题](#-常见问题)
- [🛠 开发](#-开发)

####

<br/>

</details>

## ✨ 功能特性

> \[!IMPORTANT]
>
> **Star Us** — 点一下 Star，即可第一时间收到 GitHub 的版本更新通知 \~ ⭐️

`tgup` 专注解决一个问题：**把本地媒体稳定上传到 Telegram 收藏夹**。核心目标：

- **🎯 可预测** — 扫描、分组、排序、上传，计划清晰
- **🔄 可恢复** — SQLite 状态库 + 断点续传
- **🤝 可协同** — 默认 FIFO 排队，避免多终端互抢
- **⚡ 可并行** — 强制并行模式，隔离 state/session

<table>
<tr><th>功能</th><th>默认行为</th><th>相关参数</th><th>说明</th></tr>
<tr><td>🔐 登录（验证码/QR）</td><td>复用 session</td><td><code>login --code</code> / <code>login --qr</code></td><td>支持 2FA</td></tr>
<tr><td>🧪 快速验证</td><td>生成测试图 → 上传 → 清理</td><td><code>demo</code></td><td>一条命令完成端到端验证</td></tr>
<tr><td>📂 媒体扫描</td><td>递归扫描、不过软链</td><td><code>--src</code> <code>--recursive</code> <code>--include-ext</code></td><td>支持多个 <code>--src</code>，按真实路径去重</td></tr>
<tr><td>📑 分组与切片</td><td>按父目录分组，按 10 切片</td><td><code>--order</code> <code>--album-max</code></td><td>Telegram album 上限 10</td></tr>
<tr><td>🚀 Album 并发</td><td>5 个 album 并行</td><td><code>--concurrency-album</code></td><td>并发单位是 album，不是单文件</td></tr>
<tr><td>🧩 分片上传</td><td>单文件 8 线程</td><td><code>--threads</code></td><td>512 KB 分片 + 多线程并行</td></tr>
<tr><td>🔌 DC 连接池</td><td>8 条 MTProto 连接</td><td><code>--pool-size</code></td><td>突破单连接带宽上限</td></tr>
<tr><td>💾 断点续传</td><td>默认开启</td><td><code>--resume</code> / <code>--no-resume</code></td><td>以 <code>path + size + mtime_ns</code> 识别</td></tr>
<tr><td>📋 重复策略</td><td><code>ask</code></td><td><code>--duplicate {skip,ask,upload}</code></td><td>仅在 resume 开启时生效</td></tr>
<tr><td>🔒 队列协调</td><td>同一 state FIFO 排队</td><td>默认模式</td><td>跨进程 SQLite <code>run_queue</code></td></tr>
<tr><td>⚡ 强制并行</td><td>关闭</td><td><code>--force-multi-command</code></td><td>自动隔离 state/session</td></tr>
<tr><td>🧹 状态维护</td><td>默认开启</td><td><code>--maintenance</code> <code>--cleanup-now</code></td><td>按时/按体积/按行数触发清理</td></tr>
<tr><td>📊 上传进度</td><td>打开</td><td><code>--no-progress</code></td><td>总字节 + album/files 双行进度</td></tr>
<tr><td>📝 计划预览</td><td>关闭</td><td><code>--plan</code> <code>--plan-files</code></td><td>上传前打印分组计划</td></tr>
<tr><td>🌐 MCP HTTP 服务</td><td>关闭</td><td><code>mcp serve</code></td><td>Streamable HTTP + SSE</td></tr>
</table>

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📋 Telegram 约束

> \[!NOTE]
>
> 以下是 Telegram 平台限制，非 tgup 限制。

- 单个 album 最多 **10** 个媒体（硬限制）
- album 只有 **1** 条 caption（通常给第一个媒体）
- 仅支持 MTProto 客户端登录，**不是** Bot API

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📦 安装

- **Go** >= 1.25
- 编译为单一可执行文件，无外部运行时依赖

从 [GitHub Releases][github-release-link] 下载预编译二进制（Linux / macOS / Windows，amd64 / arm64），或从源码构建：

```bash
go build -o tgup ./cmd/tgup
```

> \[!TIP]
>
> - 视频元数据识别依赖系统 `ffprobe`（FFmpeg 的一部分）。若缺失，Telegram 侧视频可能显示 `duration=0 / 1x1`。
> - tgup 预检会检测 `ffprobe` 可用性，不可用会输出告警。
> - SQLite 使用纯 Go 实现（`modernc.org/sqlite`），无需 CGo。

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🚀 快速开始

**1.** 在 [my.telegram.org](https://my.telegram.org) 获取 `api_id` 和 `api_hash`

**2.** 登录并生成 session：

```bash
tgup login --code
# 或
tgup login --qr
```

**3.** （可选）快速验证登录和上传是否正常：

```bash
tgup demo
```

**4.** 先看上传计划：

```bash
tgup dry-run --src /path/to/media --order mtime
```

**5.** 开始上传：

```bash
tgup run --src /path/to/media --caption "daily"
```

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ⌨️ 命令说明

### 顶层

```bash
tgup [-h] {login,dry-run,run,demo,mcp,version}
```

### `demo`

```bash
tgup demo [--config CONFIG] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION]
```

快速验证：生成 2 张测试图片 → 上传到 Saved Messages → 自动清理。一条命令完成端到端验证。

### `login`

```bash
tgup login [--config CONFIG] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION] (--qr | --code) [--phone PHONE]
```

| 参数 | 说明 |
|------|------|
| `--qr` | 终端打印二维码登录 |
| `--code` | 验证码登录，必要时提示 2FA 密码 |
| `--session` | session 文件路径（默认 `./secrets/session.session`） |

### `dry-run`

```bash
tgup dry-run [--src SRC] [--recursive] [--follow-symlinks] [--include-ext CSV] [--exclude-ext CSV] [--order {name,mtime,size,random}] [--reverse] [--album-max N]
```

只扫描 + 构建计划，不上传。打印文件总数、图片/视频数量、album 列表。

### `run`

```bash
tgup run [--src SRC] [--caption CAPTION] [--concurrency-album N] [--threads N] [--pool-size N] [--duplicate {skip,ask,upload}] [--force-multi-command] [--plan] ...
```

常用参数：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--target` | `me` | 上传目标 |
| `--parse-mode` | `plain` | `plain` 或 `md` |
| `--concurrency-album` | `5` | album 级并发数 |
| `--threads` | `8` | 单文件并行分片线程数（512 KB/片） |
| `--pool-size` | `8` | DC 连接池大小（`0` 禁用） |
| `--strict-metadata` | 关闭 | 视频元数据异常时拒绝上传 |
| `--image-mode` | `auto` | 图片发送策略 |
| `--video-thumbnail` | `auto` | 视频封面策略 |
| `--state` | `./data/state.sqlite` | SQLite 状态库路径 |
| `--artifacts-dir` | `./data/runs` | 运行产物根目录 |
| `--duplicate` | `ask` | `ask` / `skip` / `upload` |
| `--plan` | 关闭 | 上传前打印计划 |

### `mcp`

```bash
tgup mcp serve [--host HOST] [--port PORT] [--token TOKEN] [--allow-root PATH ...] [--enable-sse]
tgup mcp schema --out /path/to/schema.json
```

- `mcp serve` — 启动本地 MCP Streamable HTTP 服务，默认 `127.0.0.1:8765`
- `mcp schema` — 导出所有 MCP 工具的 JSON Schema
- 所有 `/mcp` 请求需要 `Bearer token`
- `GET /mcp` + `Accept: text/event-stream` — 订阅 SSE 事件流（支持 `Last-Event-ID`）

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🔀 运行模式

### 默认队列模式（推荐）

多个终端同时执行 `tgup run`，只要共享同一个 `--state`：

- 自动进入统一 FIFO 队列
- 只有队首任务进入上传
- 其他任务等待并输出 `waiting ahead=N`

适合：同一套素材库 + 同一份断点状态。

### 强制并行模式

```bash
tgup run --src /path/to/media --force-multi-command
```

- 跳过全局队列
- 自动派生隔离的 state/session（`.force.<pid>.<ts>` 后缀）
- 不共享断点状态
- 自动关闭维护任务

适合：多批任务并行跑，互不影响。

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ⚙️ 配置系统

**优先级：** `CLI > ENV > 配置文件 > 默认值`

### 配置文件来源

| 来源 | 路径 |
|------|------|
| 全局 | `~/.config/tgup/config.toml` |
| 项目 | `./tgup.toml`（覆盖全局同名字段） |
| 显式指定 | `--config /path/to/config.toml`（只使用这一份） |

> \[!NOTE]
>
> 配置文件里的相对路径按**配置文件所在目录**解析，不按当前 shell 目录解析。
> 涉及字段：`telegram.session`、`paths.state`、`scan.src`、`mcp.allow_roots`、`mcp.control_db`。

完整配置模板：[`tgup.example.toml`](tgup.example.toml)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🌐 环境变量

<details>
<summary><kbd>基础</kbd></summary>

| 变量 | 说明 |
|------|------|
| `TGUP_API_ID` | Telegram API ID |
| `TGUP_API_HASH` | Telegram API Hash |
| `TGUP_SESSION` / `TGUP_SESSION_PATH` | session 路径（互斥） |
| `TGUP_STATE` / `TGUP_STATE_PATH` | 状态库路径（互斥） |
| `TGUP_ARTIFACTS_DIR` | 产物根目录 |
| `TGUP_THREADS` | 单文件上传线程数 |
| `TGUP_POOL_SIZE` | DC 连接池大小 |

</details>

<details>
<summary><kbd>维护配置</kbd></summary>

| 变量 | 说明 |
|------|------|
| `TGUP_MAINTENANCE_ENABLED` | 启用维护 |
| `TGUP_MAINTENANCE_INTERVAL_HOURS` | 清理间隔 |
| `TGUP_MAINTENANCE_RETENTION_SENT_DAYS` | 已发送记录保留天数 |
| `TGUP_MAINTENANCE_RETENTION_FAILED_DAYS` | 失败记录保留天数 |
| `TGUP_MAINTENANCE_RETENTION_QUEUE_DAYS` | 队列记录保留天数 |
| `TGUP_MAINTENANCE_MAX_DB_MB` | DB 体积触发阈值 |
| `TGUP_MAINTENANCE_MAX_UPLOAD_ROWS` | 行数触发阈值 |
| `TGUP_MAINTENANCE_FIRST_RUN_PREVIEW` | 首次预览 |
| `TGUP_MAINTENANCE_VACUUM_COOLDOWN_HOURS` | VACUUM 冷却 |
| `TGUP_MAINTENANCE_VACUUM_MIN_RECLAIM_MB` | VACUUM 最小回收 |

</details>

<details>
<summary><kbd>MCP 配置</kbd></summary>

| 变量 | 说明 |
|------|------|
| `TGUP_MCP_ENABLED` | 启用 MCP 服务 |
| `TGUP_MCP_HOST` | 监听地址 |
| `TGUP_MCP_PORT` | 监听端口 |
| `TGUP_MCP_TOKEN` | Bearer token |
| `TGUP_MCP_ALLOW_ROOTS` | 允许的根路径 |
| `TGUP_MCP_CONTROL_DB` | 控制库路径 |
| `TGUP_MCP_EVENT_RETENTION_HOURS` | 事件保留时间 |
| `TGUP_MCP_MAX_CONCURRENT_JOBS` | 最大并发任务数 |
| `TGUP_MCP_ENABLE_SSE` | 启用 SSE |
| `TGUP_MCP_ALLOWED_ORIGINS` | CORS 源 |

</details>

> 布尔值支持：`1/0`、`true/false`、`yes/no`、`on/off`

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🔍 关键行为

### 扫描与分组

- 按 `src_root + parent_dir` 双键分组 — 不同 `--src` 即使目录同名也不会混在同一 album
- 默认媒体扩展名：
  - 图片：`.jpg` `.jpeg` `.png` `.webp` `.heic`
  - 视频：`.mp4` `.mov` `.mkv` `.webm`

### 上传与重试

- 支持 album 和单文件上传
- `FloodWait` — 按 Telegram 给出的秒数等待后重试
- 其他异常 — 指数退避重试（带随机抖动）
- `ImageProcessFailedError` — 自动回退为 document 发送
- 上传前视频元数据预检：`duration > 0 && width > 1 && height > 1`
- 上传后校验 `DocumentAttributeVideo`，要求 `supports_streaming = true`
- 每次 run 输出产物：`data/runs/<run_id>/upload.log`、`report.json`、`report.md`

### 断点续传

- `uploads` 表主键：`(path, size, mtime_ns)`
- `status='sent'` 视为已完成，后续 run 可跳过
- `--duplicate` 只控制"已发送项是否重传"

### 维护清理

触发条件（任一满足）：

- 距上次清理超过 `interval_hours`
- DB 体积超过 `max_db_mb`
- `uploads` 行数超过 `max_upload_rows`

清理流程：首次预览 → 删除过期行 → 评估可回收空间 → 满足条件时 `VACUUM`。

`--cleanup-now` 可跳过首次预览并立即执行。

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📁 项目结构

```
cmd/tgup/           入口
internal/
  app/               命令编排与业务逻辑
  artifacts/          运行产物（日志、报告）
  cli/                参数解析与命令入口
  config/             TOML 配置加载、合并、校验
  files/              文件系统抽象、路径安全
  logging/            结构化日志与脱敏
  mcp/                MCP HTTP 控制面与 SSE
  media/              视频元数据与封面提取
  plan/               album 分组与排序
  progress/           终端上传进度渲染
  queue/              跨进程 FIFO 队列协调
  scan/               文件发现与扩展名过滤
  state/              SQLite 状态持久化与维护清理
  tg/                 Telegram 传输层（gotd 适配）
  upload/             上传编排、重试与失败处理
  xerrors/             应用级错误分类
```

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ❓ 常见问题

<details>
<summary><kbd>Q：报错 <code>missing api_id</code> / <code>missing api_hash</code></kbd></summary>

配置链路中没有拿到凭证。检查：

1. 是否传了 `--api-id` / `--api-hash`？
2. 是否设置了 `TGUP_API_ID` / `TGUP_API_HASH`？
3. `tgup.toml` 是否包含 `[telegram]` 段？

</details>

<details>
<summary><kbd>Q：为什么多终端 run 只有一个在上传？</kbd></summary>

这是默认 FIFO 协调行为，防止同一 state 并发冲突。
用 `--force-multi-command` 启用真正的并行执行。

</details>

<details>
<summary><kbd>Q：为什么第一次清理只显示预览？</kbd></summary>

默认 `first_run_preview=true`，首次触发只给出"将删除多少数据"的预览。
用 `--cleanup-now` 或设置 `first_run_preview=false` 立即执行。

</details>

<details>
<summary><kbd>Q：为什么有些图片会当成 document 发？</kbd></summary>

遇到 Telegram 的 `ImageProcessFailedError` 时会自动降级为 document 发送，属于容错设计。

</details>

<details>
<summary><kbd>Q：为什么视频在 Telegram 里显示成 0s / 1×1？</kbd></summary>

视频元数据识别失败。按顺序检查：

1. 系统是否安装了 `ffprobe`？（`ffprobe -version`）
2. 上传日志中是否出现 `precheck` / `ffprobe_missing` 告警？
3. 启用 `--strict-metadata` 可硬性阻断异常上传

</details>

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🛠 开发

```bash
# 构建
go build ./cmd/tgup

# 测试（含竞态检测）
go test -count=1 -race ./...

# 格式检查
gofmt -l .
go vet ./...
```

> \[!TIP]
>
> 更多细节见：[`AI.md`](AI.md) · [`MCP.md`](MCP.md)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

---

<details><summary><h4>📝 许可证</h4></summary>

本项目基于 [Apache License 2.0](./LICENSE) 开源。

</details>

Copyright © 2025 [babywbx][profile-link]. <br />
本项目基于 [Apache-2.0][github-license-link] 许可证开源。

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
