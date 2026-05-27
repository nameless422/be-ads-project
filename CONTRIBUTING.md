# Contributing

这份仓库当前以“业务域清晰、脚本入口统一、先验证再提交”为协作默认约定。

## 目录约定

- `be/cmd/`
  服务入口，只放启动组装代码。
- `be/internal/modules/`
  按业务域拆分，每个域内部统一为：
  - `application/`
  - `domain/`
  - `infrastructure/`
- `be/internal/shared/`
  跨业务域共享但不归属于单一模块的模型和消息定义。
- `be/scripts/`
  后端运行、基础设施和链路验收脚本，按职责拆分：
  - `dev/`
  - `ops/`
  - `verify/`
- `scripts/`
  仓库级跨前后端脚本，目前只放 Harness 检查入口。
- `docs/`
  按文档用途拆分：
  - `architecture/`
  - `runbooks/`
  - `roadmap/`
  - `archive/`

新增文件时，优先放进对应业务域，不要回到“按技术散落堆放”的结构。

## 日常开发流程

推荐先走这组命令：

```bash
make up
make harness-check
make test
make verify
make down
```

更完整的命令列表：

```bash
make help
```

## 提交前检查

提交前至少做这几项：

1. `make harness-check`
2. `make test`
3. 如果改动影响本地链路，执行 `make verify`
4. 如果改动影响 Debezium 链路，执行 `make verify-debezium`
5. 确认没有把本地密钥、临时日志、运行产物提交进仓库
6. 如果改了 DLQ / replay / retention 逻辑，补充对应 README 或 runbook
7. 如果改了 worker lease / shard 分配逻辑，至少验证 `worker_leases` 和 `shard_assignments` 有真实数据

## Harness 工作流

复杂任务按 [HARNESS.md](HARNESS.md) 走：

- 动代码前先查 `docs/harness/dev-map.md`
- 跨模块或前后端联动任务先补 SPEC
- 下游发现上游文档或方案不合格时，记录阻塞项并打回，不直接替上游改结论
- 能被机器判断的规则优先沉到 `scripts/verify/verify_harness.sh`

## 提交规范

当前默认使用轻量 `Conventional Commits` 风格，保持短标题、单一意图。

推荐格式：

```text
<type>: <summary>
```

常用类型：

- `feat`: 新功能
- `fix`: 缺陷修复
- `refactor`: 重构，不改变外部行为
- `docs`: 文档调整
- `test`: 测试或验收脚本调整
- `chore`: 构建、脚本、目录整理、依赖维护

示例：

```text
feat: add google ads real mode connector
fix: handle stale pid files in service stop script
refactor: split reporting module by application domain infrastructure
docs: clarify local bootstrap flow
chore: organize project structure
```

建议：

- 一次提交只做一类事情
- 标题尽量控制在一句话内
- 不要把“重构 + 功能修改 + 文档清理”混成一个提交，除非它们强相关且不可拆

## 配置与敏感信息

- `.env.*.example` 可以提交
- 真实 `.env`、OAuth 凭证、token、secret 不要提交
- `be/logs/`、`be/run/` 这类运行产物不要提交

## 文档更新约定

以下场景应一起补文档：

- 改了目录结构：更新 `README.md` 和 `docs/README.md`
- 改了本地运行方式：更新 `README.md`、`scripts/README.md` 和 `be/scripts/README.md`
- 改了接入流程：更新对应 `runbooks/`
- 改了长期架构方向：更新 `docs/README.md`
- 改了 Harness 流程、验证或任务状态：更新 `HARNESS.md` 或 `docs/harness/`
