# Scripts Guide

`scripts/` 按职责分组：

- `dev/`
  本地基础设施、Debezium、docker compose 相关脚本。
- `ops/`
  服务启动、停止、状态查看脚本。
- `verify/`
  阶段性验收和联调检查脚本。

常用顺序：

1. `./scripts/dev/dev_phase1_stack_up.sh`
2. `./scripts/ops/start.sh`
3. `./scripts/ops/status.sh`
4. `./scripts/verify/verify_phase1.sh`
