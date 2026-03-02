# tgup

高吞吐 Telegram `Saved Messages (me)` 媒体上传 CLI。
基于 gotd（MTProto），面向图片/视频批量上传，支持断点续传、队列协调和多命令并行隔离。

License: Apache-2.0

## 1. 项目定位

`tgup` 解决的是"把本地媒体稳定上传到 Telegram 收藏夹（Saved Messages）"这个场景，核心目标是：

- 上传行为可预测（扫描、分组、排序清晰）
- 失败可恢复（SQLite 状态库 + 断点续传）
- 多终端可协同（默认 FIFO 排队，避免互抢）
- 在需要时可强制并行（隔离 state/session）

## 2. 功能清单（逐项）

| 功能 | 默认行为 | 相关参数 | 关键说明 |
|---|---|---|---|
| 登录（验证码/QR） | 复用 session 文件 | `login --code` / `login --qr` / `--session` | 支持 2FA；session 文件是敏感凭证 |
| 快速验证（demo） | 生成测试图 → 上传 → 清理 | `demo` | 一条命令验证登录和上传链路 |
| 扫描媒体 | 递归扫描、不过软链 | `--src` `--recursive` `--follow-symlinks` `--include-ext` `--exclude-ext` | 支持多个 `--src`，会去重真实路径 |
| 分组与切片 | 按父目录分组，按 10 切片 | `--order` `--reverse` `--album-max` | Telegram album 上限 10 |
| 上传并发 | album 级并发 5 | `--concurrency-album` | 并发单位是 album，不是单文件 |
| 分片并行 | 单文件 8 线程并行上传 | `--threads` | 512 KB 分片 + 多线程并行，加速大文件 |
| 断点续传 | 默认开启 | `--resume/--no-resume` | 以 `path + size + mtime_ns` 识别是否已发送 |
| 重复文件策略 | `ask` | `--duplicate {skip,ask,upload}` | 仅在 `resume` 开启时生效 |
| 多命令协调 | 同一 state FIFO 排队 | 默认模式 | 跨进程共享 SQLite `run_queue` |
| 强制并行模式 | 关闭全局排队 | `--force-multi-command` | 自动隔离 state/session，不共享续传记录 |
| 状态库维护 | 默认开启 | `--maintenance` `--cleanup-now` | 支持首次预览、按时/按体积/按行数触发清理 |
| 上传进度 | 打开 | `--no-progress` | 双行进度：总字节 + album/files 状态 |
| 上传前计划预览 | 关闭 | `--plan` `--plan-files` | 可在 run 前打印分组计划 |
| MCP HTTP 服务 | 关闭 | `mcp serve` | Streamable HTTP + SSE，支持队列任务、状态和事件流 |

## 3. Telegram 约束（必须了解）

- 单个 album 最多 10 个媒体（硬限制）
- album 只有 1 条 caption（通常给第一个媒体）
- 本项目只支持 MTProto 客户端登录，不是 Bot API

## 4. 安装与环境

- Go: `>= 1.25`
- 编译为单一可执行文件，无外部运行时依赖

从 [GitHub Releases](https://github.com/babywbx/tgup/releases) 下载预编译二进制（支持 Linux/macOS/Windows，amd64/arm64），或从源码构建：

```bash
go build -o tgup ./cmd/tgup
```

说明：

- 视频元数据识别依赖系统 `ffprobe`（FFmpeg 的一部分）
- 若 `ffprobe` 不可用，Telegram 侧视频可能出现 `duration=0 / w=1 / h=1` 的异常展示
- `tgup` 预检会检测 `ffprobe` 可用性，不可用会输出告警
- SQLite 使用纯 Go 实现（`modernc.org/sqlite`），无需 CGo

## 5. 快速开始

1. 在 [my.telegram.org](https://my.telegram.org) 获取 `api_id` 和 `api_hash`
2. 登录并生成 session：

```bash
tgup login --code
# 或
tgup login --qr
```

3. （可选）快速验证登录和上传是否正常：

```bash
tgup demo
```

4. 先看上传计划：

```bash
tgup dry-run --src /path/to/media --order mtime
```

5. 开始上传：

```bash
tgup run --src /path/to/media --caption "daily"
```

## 6. 命令说明

### 6.1 顶层

```bash
tgup [-h] {login,dry-run,run,demo,mcp,version}
```

### 6.2 `demo`

```bash
tgup demo [--config CONFIG] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION]
```

- 快速验证登录和上传链路：生成 2 张测试图片 → 上传到 Saved Messages → 自动清理
- 不需要准备任何媒体文件，一条命令完成端到端验证

### 6.3 `login`

```bash
tgup login [--config CONFIG] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION] (--qr | --code) [--phone PHONE]
```

- `--qr`: 终端打印二维码登录
- `--code`: 验证码登录，必要时会提示 2FA 密码
- `--session`: session 文件路径（默认 `./secrets/session.session`）

### 6.4 `dry-run`

```bash
tgup dry-run [--config CONFIG] [--src SRC] [--recursive|--no-recursive] [--follow-symlinks|--no-follow-symlinks] [--include-ext CSV] [--exclude-ext CSV] [--order {name,mtime,size,random}] [--reverse|--no-reverse] [--album-max N]
```

- 只扫描 + 构建计划，不上传
- 会打印：文件总数、图片/视频数量、album 列表、文件名摘要

### 6.5 `run`

```bash
tgup run [--config CONFIG] [--src SRC] [--recursive|--no-recursive] [--follow-symlinks|--no-follow-symlinks] [--include-ext CSV] [--exclude-ext CSV] [--order {name,mtime,size,random}] [--reverse|--no-reverse] [--album-max N] [--api-id API_ID] [--api-hash API_HASH] [--session SESSION] [--target TARGET] [--caption CAPTION] [--parse-mode {plain,md}] [--concurrency-album N] [--threads N] [--strict-metadata|--no-strict-metadata] [--image-mode {auto,photo,document}] [--video-thumbnail {auto,off}] [--state STATE] [--artifacts-dir DIR] [--resume|--no-resume] [--maintenance|--no-maintenance] [--cleanup-now] [--duplicate {skip,ask,upload}] [--force-multi-command] [--no-progress] [--plan] [--plan-files]
```

常用参数：

- `--target`: 默认 `me`
- `--parse-mode`: `plain`（默认）或 `md`
- `--concurrency-album`: album 并发数，默认 `5`
- `--threads`: 单文件并行分片上传线程数，默认 `8`（分片大小固定 512 KB）
- `--strict-metadata`: 视频元数据异常时拒绝上传该 album（默认关闭）
- `--image-mode`: 图片发送策略，默认 `auto`
- `--video-thumbnail`: 视频封面策略，默认 `auto`
- `--artifacts-dir`: 每次 run 的产物根目录，默认 `./data/runs`
- `--state`: SQLite 状态库路径，默认 `./data/state.sqlite`
- `--duplicate`:
  - `ask`（默认）: 询问是否重传已发送文件
  - `skip`: 跳过已发送文件
  - `upload`: 继续重传重复文件
- `--plan`: 上传前先打印计划（简版）
- `--plan-files`: 计划中包含文件名

### 6.6 `mcp`

```bash
tgup mcp serve [--config CONFIG] [--host HOST] [--port PORT] [--token TOKEN] [--allow-root PATH ...] [--control-db PATH] [--event-retention-hours HOURS] [--max-concurrent-jobs N] [--enable-sse|--no-enable-sse]
tgup mcp schema --out /path/to/schema.json
```

- `mcp serve`: 启动本地 MCP Streamable HTTP 服务，默认监听 `127.0.0.1:8765`
- `mcp schema`: 导出所有 MCP 工具的 JSON Schema 契约
- `Authorization`: 所有 `/mcp` 请求要求 `Bearer token`
- `GET /mcp` + `Accept: text/event-stream`: 订阅 SSE 事件流（支持 `Last-Event-ID`）

## 7. 运行模式

### 7.1 默认队列模式（推荐）

多个终端同时执行 `tgup run`，只要共享同一个 `--state`：

- 自动进统一 FIFO 队列
- 只有队首任务进入上传
- 其他任务等待并输出 `waiting ahead=N`

适合"同一套素材库 + 同一份断点状态"场景。

### 7.2 强制并行模式

```bash
tgup run --src /path/to/media --force-multi-command
```

行为变化：

- 跳过全局队列
- 自动派生隔离的 state/session 文件（带 `.force.<pid>.<ts>` 后缀）
- 不共享断点状态
- 自动关闭维护任务（避免对临时 state 做清理）

适合"多批任务并行跑，互不影响"的场景。

## 8. 配置系统

优先级：

`CLI > ENV > config 文件 > 默认值`

### 8.1 配置文件来源

- 全局：`~/.config/tgup/config.toml`
- 项目：`./tgup.toml`（覆盖全局同名字段）
- 显式：`--config /path/to/config.toml`（只使用这一份）

### 8.2 路径解析规则

配置文件里的相对路径按"配置文件所在目录"解析，不按当前 shell 目录解析。
涉及字段：`telegram.session`、`paths.state`、`scan.src`、`mcp.allow_roots`、`mcp.control_db`。

### 8.3 完整配置模板

见 [`tgup.example.toml`](tgup.example.toml)。

## 9. 环境变量

基础：

- `TGUP_API_ID`
- `TGUP_API_HASH`
- `TGUP_SESSION` / `TGUP_SESSION_PATH`（互斥，不能同时设置不同值）
- `TGUP_STATE` / `TGUP_STATE_PATH`（互斥，不能同时设置不同值）
- `TGUP_ARTIFACTS_DIR`
- `TGUP_THREADS`

维护配置：

- `TGUP_MAINTENANCE_ENABLED`
- `TGUP_MAINTENANCE_INTERVAL_HOURS`
- `TGUP_MAINTENANCE_RETENTION_SENT_DAYS`
- `TGUP_MAINTENANCE_RETENTION_FAILED_DAYS`
- `TGUP_MAINTENANCE_RETENTION_QUEUE_DAYS`
- `TGUP_MAINTENANCE_MAX_DB_MB`
- `TGUP_MAINTENANCE_MAX_UPLOAD_ROWS`
- `TGUP_MAINTENANCE_FIRST_RUN_PREVIEW`
- `TGUP_MAINTENANCE_VACUUM_COOLDOWN_HOURS`
- `TGUP_MAINTENANCE_VACUUM_MIN_RECLAIM_MB`

MCP 配置：

- `TGUP_MCP_ENABLED`
- `TGUP_MCP_HOST`
- `TGUP_MCP_PORT`
- `TGUP_MCP_TOKEN`
- `TGUP_MCP_ALLOW_ROOTS`
- `TGUP_MCP_CONTROL_DB`
- `TGUP_MCP_EVENT_RETENTION_HOURS`
- `TGUP_MCP_MAX_CONCURRENT_JOBS`
- `TGUP_MCP_ENABLE_SSE`
- `TGUP_MCP_ALLOWED_ORIGINS`

布尔值支持：`1/0`、`true/false`、`yes/no`、`on/off`。

## 10. 关键行为细节

### 10.1 扫描与分组

- 按 `src_root + parent_dir` 双键分组
- 不同 `--src` 即使目录同名，也不会混在同一 album
- 默认媒体扩展名：
  - image: `.jpg .jpeg .png .webp .heic`
  - video: `.mp4 .mov .mkv .webm`

### 10.2 上传与重试

- album 和单文件都支持上传
- `FloodWait` 按 Telegram 给出的秒数等待后重试
- 其他异常走指数退避重试（带随机抖动）
- `ImageProcessFailedError` 会自动回退为 document 发送
- 上传前会对视频做元数据预检，要求 `duration > 0 && width > 1 && height > 1`
- 上传后会校验 Telegram 返回的 `DocumentAttributeVideo`，并要求 `supports_streaming = true`
- 每次 run 会输出产物目录：`data/runs/<run_id>/upload.log`、`report.json`、`report.md`

### 10.3 断点续传

- `uploads` 表主键是 `(path, size, mtime_ns)`
- `status='sent'` 会被视为已完成，后续 run 可跳过
- `--duplicate` 只控制"已发送项是否重传"

### 10.4 维护清理

触发条件（任一满足）：

- 距上次清理超过 `interval_hours`
- DB 体积超过 `max_db_mb`
- `uploads` 行数超过 `max_upload_rows`

清理流程：

1. 首次触发可先预览（`first_run_preview=true`）
2. 真正执行时删除过期 sent/failed/run_queue 行
3. 评估可回收空间；满足阈值和冷却条件时执行 `VACUUM`

`--cleanup-now` 可跳过首次预览并立即执行一次。

## 11. 目录结构

```text
cmd/tgup/           入口
internal/
  app/               命令编排与业务逻辑串联
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
  xerrors/            应用级错误分类
```

## 12. 常见问题

### Q1: 报错 `missing api_id` / `missing api_hash`

说明最终配置链路里没有拿到凭证。请检查：

1. 是否传了 `--api-id/--api-hash`
2. 是否设置了 `TGUP_API_ID/TGUP_API_HASH`
3. `tgup.toml` 是否含 `[telegram]` 段

### Q2: 为什么多终端 run 只有一个在上传？

这是默认 FIFO 协调行为，防止同一 state 并发冲突。
如果你明确要并发，请使用 `--force-multi-command`。

### Q3: 为什么第一次清理只显示预览？

默认 `first_run_preview=true`，首次触发只给出"将删除多少数据"的预览。
要立即执行可使用 `--cleanup-now` 或把 `first_run_preview=false`。

### Q4: 为什么有些图片会当成 document 发？

遇到 Telegram 的 `ImageProcessFailedError` 时会自动降级为 document 发送，属于容错设计。

### Q5: 为什么视频在 Telegram 里显示成 0s / 1x1？

通常是发送时视频元数据识别失败。可按顺序检查：

1. 系统是否安装了 `ffprobe`（`ffprobe -version`）
2. 上传日志中是否出现 `precheck` / `ffprobe_missing` 告警
3. 需要硬性阻断异常时，启用 `--strict-metadata`

## 13. 开发与质量

```bash
# 构建
go build ./cmd/tgup

# 测试（含竞态检测）
go test -count=1 -race ./...

# 格式检查
gofmt -l .
go vet ./...
```

更多细节见：

- `AI.md`
- `MCP.md`
