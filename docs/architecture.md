# Architecture

This repository follows a transport-isolated design:

1. `internal/tg` is the only package allowed to know Telegram transport details.
2. `internal/upload` owns business rules and retries.
3. `internal/state` and `internal/queue` preserve resumability and process coordination.
4. `internal/mcp` owns the HTTP control plane.

The target is long-term maintainability with explicit boundaries and testable deterministic modules.

