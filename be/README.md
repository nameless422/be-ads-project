# be

后端代码和本地运行入口放在这里。前端 React 工程在 `../fe`，`bi-api` 会从 `../fe/dist` 读取 `/bi` 页面构建产物。

## 目录

- `cmd/`: `bi-api`、`control-plane`、`collector-worker`、`transformer-worker` 等服务入口
- `internal/`: collection、transformation、reporting、controlplane 等业务模块
- `scripts/dev/`: 本地 MySQL / ClickHouse / NATS / Debezium 环境
- `scripts/ops/`: 服务启动、停止、状态查看
- `scripts/verify/`: 本地主链路和 Debezium 链路验收
- `deploy/`: 部署和观测配置
- `docker-compose.dev.yml`: 本地依赖服务

## 常用命令

```bash
make up
make status
make verify
make test
make down
```

Mac 新环境首次启动可以直接用：

```bash
make mac-start
```

这个入口会检查 Xcode Command Line Tools、Homebrew、Go、Node.js、Docker Desktop、本地端口，安装缺失的 brew 依赖，然后启动并验收本地栈。

前端相关命令也从 `be/` 执行：

```bash
make frontend-install
make frontend-build
make frontend-dev
```

等价于操作 `../fe` 目录下的 Vite 项目。

## 保留脚本

```text
scripts/dev/dev_base_stack_up.sh
scripts/dev/dev_base_stack_down.sh
scripts/dev/dev_debezium_up.sh
scripts/dev/dev_debezium_down.sh
scripts/ops/up.sh
scripts/ops/down.sh
scripts/ops/start.sh
scripts/ops/stop.sh
scripts/ops/status.sh
scripts/ops/mac_local_start.sh
scripts/verify/common.sh
scripts/verify/verify_local_stack.sh
scripts/verify/verify_debezium_pipeline.sh
```

其他临时包装脚本、旧 runbook、截图和运行产物已经清掉，避免根目录继续膨胀。

## SDD

- `openspec/`: 轻量 OpenSpec，用来记录后端 proposal、design、tasks 和长期能力 spec。
- `specs/`: Spec Kit 风格的 feature spec、plan、tasks、quickstart。
- `.specify/`: Spec Kit 项目约束和长期约定。

后端 SDD 覆盖 Go 服务、API、数据链路、存储、消息和本地运维入口。React 页面和前端交互 SDD 放在 `../fe`。
