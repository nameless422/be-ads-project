# be_ads_project Harness

这份文件是本仓库的 Harness 入口。它把项目级上下文、工作流边界、验证入口和持续维护规则放进仓库，而不是只留在对话记忆里。

## 先读什么

1. [docs/harness/README.md](docs/harness/README.md)
   Harness 总览，以及每类资产负责什么。
2. [AGENTS.md](AGENTS.md)
   给 AI 执行者的根规则：先读什么、不能碰什么、收尾跑什么。
3. [docs/harness/dev-map.md](docs/harness/dev-map.md)
   动代码前先查这里，确认功能落点、影响范围和已有写法。
4. [docs/harness/playbook.md](docs/harness/playbook.md)
   后续开发需求怎么接、怎么拆、怎么让 Codex 使用、怎么验证。
5. [docs/harness/sdd.md](docs/harness/sdd.md)
   SDD 和 Harness 的分工、存放位置、更新规则和当前 SDD 索引。
6. [docs/harness/workflow.md](docs/harness/workflow.md)
   从需求到验证的阶段流转、角色边界、回退规则和交接边界。
7. [docs/harness/workflow.json](docs/harness/workflow.json)
   可机器解析的流程阶段定义，供脚本和后续自动化检查使用。
8. [docs/harness/task-board.md](docs/harness/task-board.md)
   当前任务索引、阶段、文档入口和交付结论。
9. [docs/harness/tasks/_template](docs/harness/tasks/_template)
   复杂任务的阶段文档模板，复制后放到 `docs/harness/tasks/YYYYMMDD-<slug>/`。

## 完成定义

一次改动默认满足下面条件才算完成：

1. 需求边界明确，复杂需求先补 SPEC，可用 [docs/harness/tasks/_template/01-spec.md](docs/harness/tasks/_template/01-spec.md)。
2. 改代码前查过 dev-map，确认落点不是重新发明一套平行结构。
3. 下游发现上游文档或方案有问题时，只记录阻塞项并打回，不直接替上游改结论。
4. 改动涉及结构、命令、接口、页面或运行方式时，同步更新 README、docs、scripts 入口或 task-board。
5. 改动涉及长期能力设计时，同步更新对应 SDD：后端放 `be/openspec` 或 `be/specs`，前端放 `fe/openspec` 或 `fe/specs`。
6. 至少执行 `make harness-check`。影响本地主链路时继续执行 `make verify`，影响 Debezium 时继续执行 `make verify-debezium`。

## Harness 和 SDD 整合方式

本仓库采用“整合流程、分开产物”的方式。Harness 是唯一需求入口和执行轨道，SDD 是被 Harness 在需求分析、方案设计和收口阶段调用的设计档案。完整规则见 [docs/harness/sdd.md](docs/harness/sdd.md)。不要让 SDD 自己形成另一套任务流，也不要把所有阶段结论塞进一份巨型文档：

| 内容 | 放哪里 | 什么时候更新 |
| --- | --- | --- |
| AI/人协作规则、交付定义 | `AGENTS.md`、`HARNESS.md` | 流程规则变化 |
| 需求阶段、验证结果、阻塞项 | `docs/harness/tasks/`、`docs/harness/task-board.md` | 复杂任务推进或收口 |
| 后端接口、数据链路、存储、worker 设计 | `be/openspec`、`be/specs` | 后端能力、字段或链路变化 |
| 前端页面、交互、路由、API client 设计 | `fe/openspec`、`fe/specs` | 前端能力或页面结构变化 |
| 新人入口和命令索引 | `README.md`、`be/README.md`、`fe/README.md` | 目录、命令或主入口变化 |

判断规则：

- 只影响一次任务推进：写 Harness task。
- 会影响后续开发理解：写 SDD。
- 两者都影响：Harness 记录阶段和验证，SDD 记录长期设计。

## 项目适配边界

本项目是前后端仓库。后端源码、Go module、vendor、运行脚本和后端配置都在 `be/`；前端源码在 `fe/package.json` 和 `fe/src/`。根目录只保留 Makefile、Harness、CI、docs 和跨前后端脚本。`fe/dist` 与 `fe/node_modules` 仍然只是生成物，不作为源码入口。

## 核心入口

```bash
make help
make harness-check
make test
make verify
make verify-debezium
```

CI 会在 push 和 pull request 上执行同一个 `make harness-check`，配置见 [.github/workflows/harness-check.yml](.github/workflows/harness-check.yml)。

协作入口：

- PR 模板：[.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md)
- 新需求模板：[.github/ISSUE_TEMPLATE/feature_request.yml](.github/ISSUE_TEMPLATE/feature_request.yml)
