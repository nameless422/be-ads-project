# Dev Map

动代码前先查这张图。它不是文件清单，而是告诉你某类改动通常从哪里进、会牵动哪些链路、收尾该跑什么验证。

## 后端主图

| 改动类型 | 主要落点 | 常见影响面 | 默认验证 |
| --- | --- | --- | --- |
| 服务启动、生命周期、日志 | `be/cmd/*/main.go`、`be/internal/platform/kratosx/`、`be/internal/logx/` | `be/scripts/ops/*`、`README.md`、`be/scripts/README.md` | `make harness-check`，必要时 `make status` |
| 调度、lease、shard | `be/internal/modules/controlplane/` | `collector-worker` 消费、`worker_leases`、`shard_assignments`、控制面 API | `make test`，影响本地链路时 `make verify` |
| 采集任务执行 | `be/internal/modules/collection/application/`、`infrastructure/collector/` | connector、raw store、outbox、checkpoint | `make test`，影响数据链路时 `make verify` |
| 平台 connector | `be/internal/modules/collection/infrastructure/connectors/` | `be/internal/shared/ads/`、normalizer、Google Ads runbook | `make test`，真实平台切换按 runbook 验证 |
| raw store / outbox | `be/internal/modules/collection/infrastructure/rawstore/mysql/` | Debezium、JetStream、transformer-worker、verify scripts | `make test`，必要时 `make verify-debezium` |
| 标准化转换 | `be/internal/modules/transformation/` | serving mysql、clickhouse、BI 查询字段 | `make test`，影响报表时 `make verify` |
| BI 查询和 HTTP API | `be/internal/modules/reporting/`、`be/cmd/bi-api/` | 前端调用、curl 示例、字段文档 | `make test`，影响接口时 `make verify` |
| 共享广告模型 | `be/internal/shared/ads/` | collection、transformation、reporting 三层 | `make test`，必要时同步字段 lineage |
| 消息模型 / JetStream | `be/internal/shared/messaging/`、各模块 `infrastructure/jetstream/` | control-plane、collector、transformer、Debezium | `make test`，必要时 `make verify-debezium` |
| 本地基础设施 | `be/docker-compose.dev.yml`、`be/scripts/dev/` | MySQL、ClickHouse、NATS、Debezium | `make infra-up`，`make verify` |
| 后端验证脚本 | `be/scripts/verify/` | Makefile、Contributing、backend validation | `bash -n be/scripts/verify/*.sh`，`make harness-check` |
| 仓库级验证脚本 | `scripts/verify/` | Harness validation、CI、PR gate | `bash -n scripts/verify/*.sh`，`make harness-check` |
| 文档入口 | `README.md`、`be/README.md`、`fe/README.md`、`docs/harness/` | 新人入口、AI 上下文、任务交接 | `make harness-check` |

## 前端主图

当前仓库的前端源码在 `fe/package.json`、`fe/src/`、`fe/vite.config.ts` 和 `fe/tsconfig.json`。`fe/dist` 和 `fe/node_modules` 不是源码权威。

| 改动类型 | 预期落点 | 常见影响面 | 默认验证 |
| --- | --- | --- | --- |
| API client | `fe/src/api/` | `bi-api` 路由、接口字段、错误处理 | `make frontend-build`，再跑 `make harness-check` |
| 页面和路由 | `fe/src/pages/`、`fe/src/App.*` | BI 查询、控制面、筛选条件 | 前端构建 + 浏览器走通主要页面 |
| 组件和样式 | `fe/src/components/`、`fe/src/styles/` | 移动端布局、数据密度、可读性 | 前端构建 + 关键视口截图 |
| 构建配置 | `fe/package.json`、`fe/vite.config.ts`、`fe/tsconfig.json` | 静态资源路径、代理、部署产物 | `make frontend-build`，`make harness-check` |

不要手改 `fe/dist` 来代表源码变更。除非任务明确是修复已发布静态产物，否则应恢复或修改 `fe/src` 后重新构建。

## 常用入口

```bash
make help
make harness-check
make test
make verify
make verify-debezium
```

## 文档联动

- 改运行方式：同步 `README.md`、`scripts/README.md`、`be/scripts/README.md`、`docs/harness/playbook.md`。
- 改字段：同步 `be/README.md` 或对应 `be/openspec/` / `be/specs/`。
- 改 Google Ads 真实接入：同步 `be/README.md` 和对应后端 SDD 文档。
- 改任务阶段或交付结论：同步 `docs/harness/task-board.md`。
