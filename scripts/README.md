# Scripts Guide

`scripts/` 按职责分组：

- `dev/`
  本地基础设施和 Debezium 启动脚本。
- `ops/`
  服务启动、停止、状态查看脚本。
- `verify/`
  本地主链路和联调检查脚本。

常用顺序：

1. `./scripts/ops/up.sh`
2. `./scripts/ops/status.sh`
3. `./scripts/verify/verify_local_stack.sh`
4. `./scripts/ops/down.sh`

如果你想分步调试：

1. `./scripts/dev/dev_base_stack_up.sh`
2. `./scripts/ops/start.sh`
3. `./scripts/ops/status.sh`
4. `./scripts/verify/verify_local_stack.sh`
