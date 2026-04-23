# Contributing

这份仓库当前以“业务域清晰、脚本入口统一、先验证再提交”为协作默认约定。

## 目录约定

- `cmd/`
  服务入口，只放启动组装代码。
- `internal/modules/`
  按业务域拆分，每个域内部统一为：
  - `application/`
  - `domain/`
  - `infrastructure/`
- `internal/shared/`
  跨业务域共享但不归属于单一模块的模型和消息定义。
- `scripts/`
  按职责拆分：
  - `dev/`
  - `ops/`
  - `verify/`
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

1. `make test`
2. 如果改动影响本地链路，执行 `make verify`
3. 如果改动影响 Debezium 链路，执行 `make verify-debezium`
4. 确认没有把本地密钥、临时日志、运行产物提交进仓库
5. 如果改了 DLQ / replay / retention 逻辑，补充对应 README 或 runbook
6. 如果改了 worker lease / shard 分配逻辑，至少验证 `worker_leases` 和 `shard_assignments` 有真实数据

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
- `logs/`、`run/` 这类运行产物不要提交

## 文档更新约定

以下场景应一起补文档：

- 改了目录结构：更新 `README.md` 和 `docs/README.md`
- 改了本地运行方式：更新 `README.md` 和 `scripts/README.md`
- 改了接入流程：更新对应 `runbooks/`
- 改了长期架构方向：更新 `docs/README.md`
