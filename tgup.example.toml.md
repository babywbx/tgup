# `tgup.example.toml` 配置说明（中文）

这个文档对应 `tgup.example.toml`。
示例文件本身使用英文注释，便于直接复制到项目里使用。

## 1. 使用方式

1. 复制示例文件并改名为你的配置文件（如 `tgup.toml`）。
2. 先填写 `[telegram]` 的 `api_id` 和 `api_hash`。
3. 按需补充 `scan/plan/upload/maintenance/mcp`。
4. 运行时仍可用 CLI 参数覆盖配置值（优先级最高）。

配置优先级：

`CLI > ENV > 配置文件 > 默认值`

## 2. section 作用

- `[telegram]`：Telegram 登录凭据与会话文件路径。
- `[paths]`：上传状态库（SQLite）与产物目录路径。
- `[scan]`：扫描哪些目录、是否递归、扩展名过滤。
- `[plan]`：排序方式、排序方向、album 切片大小。
- `[upload]`：上传目标、caption、并发、断点续传。
- `[maintenance]`：状态库自动清理策略。
- `[mcp]`：MCP HTTP 服务配置。

## 3. 字段说明要点

- `scan.src`：如果不传 `--src`，这里必须提供。
- `plan.album_max`：Telegram album 限制是 `1..10`。
- `upload.parse_mode`：只支持 `plain` 或 `md`。
- `upload.state`：旧字段，建议使用 `[paths].state`。
- `maintenance.*`：用于控制 state DB 清理触发条件和保留天数。
- `mcp.token` 与 `mcp.allow_roots`：启动 `tgup mcp serve` 时是运行时必填。
- `mcp.allowed_origins`：浏览器访问 MCP 时的 Origin 白名单。

## 4. 全量字段速查（与 `tgup.example.toml` 一一对应）

| 字段 | 作用 | 默认值/约束 |
|---|---|---|
| `telegram.api_id` | Telegram API ID | 必填 |
| `telegram.api_hash` | Telegram API Hash | 必填 |
| `telegram.session` | session 文件路径 | 默认 `./secrets/session.session` |
| `paths.state` | 上传状态库（SQLite）路径 | 默认 `./data/state.sqlite` |
| `paths.artifacts_dir` | 运行产物输出根目录 | 默认 `./data/runs` |
| `scan.src` | 扫描源目录/文件列表 | 未传 `--src` 时必填 |
| `scan.recursive` | 是否递归扫描目录 | 默认 `true` |
| `scan.follow_symlinks` | 是否跟随软链接 | 默认 `false` |
| `scan.include_ext` | 允许上传的扩展名集合 | 默认内置图片+视频后缀 |
| `scan.exclude_ext` | 要排除的扩展名集合 | 默认空 |
| `plan.order` | 计划排序方式 | `name/mtime/size/random`，默认 `mtime` |
| `plan.reverse` | 是否反向排序 | 默认 `false` |
| `plan.album_max` | 每个 album 的最大文件数 | `1..10`，默认 `10` |
| `upload.target` | 上传目标对话 | 默认 `"me"`（Saved Messages） |
| `upload.caption` | album caption | 可空 |
| `upload.parse_mode` | caption 解析模式 | `plain/md`，默认 `plain` |
| `upload.concurrency_album` | album 级并发数 | `>=1`，默认 `5` |
| `upload.strict_metadata` | 视频元数据异常时是否拒绝 | 默认 `false` |
| `upload.image_mode` | 图片发送策略 | `auto/photo/document`，默认 `auto` |
| `upload.video_thumbnail` | 视频封面策略 | `auto/off`，默认 `auto` |
| `upload.resume` | 是否启用断点续传过滤 | 默认 `true` |
| `upload.duplicate` | 已发送文件重传策略 | `skip/ask/upload`，默认 `ask` |
| `upload.state` | 旧版 state 路径字段 | 兼容字段，建议用 `[paths].state` |
| `maintenance.enabled` | 是否启用自动清理 | 默认 `true` |
| `maintenance.interval_hours` | 清理检查间隔（小时） | `>0`，默认 `6` |
| `maintenance.retention_sent_days` | sent 记录保留天数 | `>=0`，默认 `90` |
| `maintenance.retention_failed_days` | failed 记录保留天数 | `>=0`，默认 `30` |
| `maintenance.retention_queue_days` | queue 记录保留天数 | `>=0`，默认 `7` |
| `maintenance.max_db_mb` | DB 大小触发阈值（MB） | `>0`，默认 `256` |
| `maintenance.max_upload_rows` | 上传记录数触发阈值 | `>0`，默认 `300000` |
| `maintenance.first_run_preview` | 首次是否只预览清理 | 默认 `true` |
| `maintenance.vacuum_cooldown_hours` | VACUUM 冷却时间（小时） | `>=0`，默认 `24` |
| `maintenance.vacuum_min_reclaim_mb` | VACUUM 最小回收估算（MB） | `>=0`，默认 `32` |
| `mcp.enabled` | 是否启用 MCP 配置段 | 默认 `false` |
| `mcp.host` | MCP 监听地址 | 默认 `127.0.0.1` |
| `mcp.port` | MCP 监听端口 | `1..65535`，默认 `8765` |
| `mcp.token` | MCP Bearer Token | `mcp serve` 运行时必填 |
| `mcp.allow_roots` | MCP 可访问目录白名单 | `mcp serve` 运行时必填 |
| `mcp.control_db` | MCP 控制/事件数据库路径 | 默认 `./data/mcp.sqlite` |
| `mcp.event_retention_hours` | MCP 事件保留时长（小时） | `>0`，默认 `72` |
| `mcp.max_concurrent_jobs` | MCP 最大并发任务数 | `>=1`，默认 `4` |
| `mcp.enable_sse` | 是否启用 SSE 事件流 | 默认 `true` |
| `mcp.allowed_origins` | 浏览器 Origin 白名单 | 默认 `null` + localhost 组 |

## 5. 最精简 `tgup.toml` 示例（只放 API 两个字段）

```toml
[telegram]
api_id = 123456
api_hash = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

说明：

- 这份最简配置只满足 Telegram 凭据。
- `session` 没写时会用默认值：`./secrets/session.session`。
- 执行 `dry-run` / `run` 时，仍需要通过 CLI 传 `--src`（或在配置里补 `scan.src`）。

## 6. 路径解析规则

配置里的相对路径按"配置文件所在目录"解析，不按当前终端目录解析。
常见字段包括：

- `telegram.session`
- `paths.state`
- `paths.artifacts_dir`
- `scan.src`
- `mcp.allow_roots`
- `mcp.control_db`
