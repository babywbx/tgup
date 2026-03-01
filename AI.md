# AI 协作说明（tgup）

本文档给 AI/自动化协作者提供项目约束与实现边界，目标是减少"改动正确但不符合项目语义"的情况。

## 1. 项目目标

`tgup` 是 Telegram Saved Messages（`me`）媒体上传器，基于 MTProto（gotd）。
核心是稳定批量上传，不是社交机器人。

## 2. 不可违背的约束

- Telegram album 最多 10 个媒体，必须切片
- album 仅 1 个 caption（挂第一个媒体）
- `parse_mode` 仅支持 `plain` / `md`
- 断点续传默认开启，状态落 SQLite
- 配置优先级固定：`CLI > ENV > config > default`

## 3. 安全与隐私

- 不输出验证码、2FA 密码、session 内容
- session 文件等同凭证，应放到 `./secrets/`（已在 `.gitignore`）
- 日志里若涉及敏感串，先走脱敏（`internal/logging.redact`）

## 4. 模块职责（变更时必须保持）

- `internal/scan`: 输入源扫描与媒体过滤
- `internal/plan`: 排序、按 `src_root + parent_dir` 分组、按 `album_max` 切片
- `internal/upload`: 发送、重试、失败处理、状态标记
- `internal/state`: `uploads/run_queue/maintenance_meta` 三类状态
- `internal/queue`: 跨进程 FIFO 队列协调
- `internal/config`: TOML 读取 + 相对路径标准化 + 多来源配置合并与参数校验
- `internal/tg`: gotd 客户端和登录流程
- `internal/progress`: 终端上传进度渲染
- `internal/artifacts`: 运行产物（日志、报告）
- `internal/files`: 文件系统抽象、路径安全
- `internal/logging`: 结构化日志与脱敏
- `internal/media`: 视频元数据与封面提取
- `internal/mcp`: MCP HTTP 控制面与 SSE
- `internal/xerrors`: 应用级错误分类
- `internal/cli`: 参数解析与命令入口
- `internal/app`: 命令编排与业务逻辑串联

## 5. 变更守则

1. 新功能先确认是否应该落在现有模块，不要堆到 `internal/cli`
2. 新增配置时必须补齐：CLI 参数、ENV、配置文件、默认值、校验
3. 与上传行为相关改动必须覆盖测试（尤其是失败/重试/断点路径）
4. 文档和 help 文案保持同步，不允许"代码已改，文档未改"

## 6. MCP 相关

当前仓库已实现 MCP HTTP 服务端（`tgup mcp serve`）。
涉及协议、工具和安全边界的变更时，以 `MCP.md` 为准，并保持文档与测试同步更新。
