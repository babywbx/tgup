# MCP（远程控制）

当前状态：已实现（截至 2026-02-28）。

## 1. 入口与传输

- 启动命令：`tgup mcp serve`
- 默认监听：`127.0.0.1:8765`
- MCP endpoint：`/mcp`
- 传输：Streamable HTTP
  - `POST /mcp`：JSON-RPC 请求/响应
  - `GET /mcp` + `Accept: text/event-stream`：SSE 事件流
- 协议版本：`2025-11-25`、`2025-06-18`

不兼容旧的 2024-11-05 双端点 HTTP+SSE 模式（不会提供 `/sse`、`/messages`）。

## 2. 安全边界

- 所有 `/mcp` 请求必须携带 `Authorization: Bearer <token>`
- 默认允许 Origin：`null`、`http://localhost`、`http://127.0.0.1`
- 所有输入路径都必须落在 `allow_roots` 白名单下
- 不暴露登录能力（`tgup login` 仍走本地 CLI）

## 3. 工具清单

- `tgup.health`
- `tgup.dry_run`
- `tgup.run.start`
- `tgup.run.sync`
- `tgup.run.status`
- `tgup.run.cancel`
- `tgup.run.events`
- `tgup.schema.get`

所有工具都返回结构化 JSON，支持 JSON Schema 导出：

```bash
tgup mcp schema --out ./data/mcp-schema.json
```

## 4. 任务与事件

- `tgup.run.start`：异步提交任务，立即返回 `job_id`
- `tgup.run.sync`：同步等待任务结束或超时
- `tgup.run.events`：按 `sinceSeq` 增量拉取
- SSE 支持 `Last-Event-ID` 断线续传
- 断开 SSE 不会自动取消任务，取消需显式调用 `tgup.run.cancel`

## 5. 存储

默认 MCP 控制库：`./data/mcp.sqlite`。

包含表：

- `mcp_jobs`：任务状态与结果
- `mcp_events`：事件序列
- `mcp_sessions`：会话与最后游标

## 6. 配置

示例（`tgup.toml`）：

```toml
[mcp]
enabled = false
host = "127.0.0.1"
port = 8765
token = ""
allow_roots = ["./media", "./data", "./secrets"]
control_db = "./data/mcp.sqlite"
event_retention_hours = 72
max_concurrent_jobs = 4
enable_sse = true
# allowed_origins = ["null", "http://localhost", "http://127.0.0.1"]
```

环境变量：

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
