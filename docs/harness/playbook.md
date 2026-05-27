# Harness Playbook

这份手册回答三个问题：

1. 开发需求怎么进入这套 Harness。
2. 不同类型任务按什么固定动作执行。
3. 收尾时跑什么验证。

## Codex 怎么用

别人 clone 后，从仓库根目录打开 Codex，先跑：

```bash
make harness-check
```

新会话第一句话可以是：

```text
先阅读 AGENTS.md 和 HARNESS.md，之后按这个仓库的 Harness 工作流开发。
```

具体需求可以直接说：

```text
按 Harness 工作流帮我实现：<需求描述>
```

复杂前后端联动需求可以说：

```text
这是一个前后端联动需求，先按 docs/harness/tasks/_template 建 SPEC，再设计、实现和验证：<需求描述>
```

如果某个 Codex 环境没有自动读取根规则，把这句话发给它：

```text
请先阅读 AGENTS.md、HARNESS.md、docs/harness/playbook.md、docs/harness/dev-map.md、docs/harness/sdd.md 和 docs/harness/workflow.md，然后按里面的流程继续。
```

## 任务类型

| 类型 | 例子 | 最少动作 |
| --- | --- | --- |
| docs-only | README、runbook、Harness 文档 | 查 `task-board`，改文档，跑 `make harness-check` |
| backend-small-change | 单个接口字段、日志、配置、connector 小修 | 查 `dev-map`，改代码，跑 `make harness-check` + `make test` |
| data-pipeline-change | 采集、outbox、转换、BI 查询、启动脚本 | 建任务文档，跑 `make harness-check` + `make test` + 相关链路验证 |
| debezium-change | outbox、raw events、Debezium 配置 | 建任务文档，跑 `make harness-check` + `make test` + `make verify-debezium` |
| frontend-change | 页面样式、筛选项、文案、API client | 改 `fe/src`，跑 `make frontend-build` + `make harness-check` |
| full-stack-change | 新页面、新接口、新字段、新状态 | 建任务文档，写清 API contract、页面入口、错误态和联调方式 |
| startup-debug | 本地启动、状态、健康检查、服务日志、端口问题 | 查状态和日志，用 `/healthz` 交叉验证，必要时跑 `make verify` |

## Harness + SDD 怎么配合

本仓库的判断是：**整合流程比平行流程更好**。平行流程容易让同一个需求同时出现在 Harness task、OpenSpec 和 Spec Kit 里，后续不知道哪个是权威；完全合并目录又会让一次任务记录和长期设计混在一起。现在采用一条路线：

```text
需求 -> Harness 路由/阶段/验证 -> 必要时更新 SDD -> task-board 收口
```

默认规则：

| 场景 | Harness 动作 | SDD 动作 |
| --- | --- | --- |
| 文案、README、脚本说明 | 更新相关 README / Harness 文档，跑 `make harness-check` | 通常不需要 |
| 后端接口、字段、worker、存储、消息链路 | 查 `dev-map`，必要时建 `docs/harness/tasks/*` | 更新 `be/openspec` 或 `be/specs` |
| 前端页面、路由、交互、API client | 查前端主图，记录页面入口和验证方式 | 更新 `fe/openspec` 或 `fe/specs` |
| 前后端联动 | 建 Harness 任务文档，SPEC 写清 API contract、页面入口和验收标准 | 后端 SDD 写接口/数据契约，前端 SDD 写页面/交互契约 |
| 临时排查 | task-board 或 issue 记录结论和未验证项 | 只有沉淀为长期规则时再补 |

优先复用已有 SDD，不为每个小改动新建一套目录。已有入口：

```text
docs/harness/sdd.md
be/openspec/project.md
be/specs/003-bi-overview-business-data-foundation/spec.md
fe/openspec/project.md
fe/specs/001-react-bi-refactor/spec.md
```

## 固定动作

### docs-only

1. 查 [task-board.md](task-board.md)，确认是否已有任务。
2. 修改对应文档，并保持入口不重复。
3. 如果新增规则或入口，同步 [HARNESS.md](../../HARNESS.md) 或 [AGENTS.md](../../AGENTS.md)。
4. 执行 `make harness-check`。

### backend-small-change

1. 查 [dev-map.md](dev-map.md)，确认模块归属。
2. 改代码，避免顺手重构无关模块。
3. 执行 `make harness-check` 和 `make test`。
4. 如果影响接口或业务行为，补 `make verify`。

### data-pipeline-change

1. 复制 [tasks/_template](tasks/_template) 建阶段文档。
2. 在 SPEC 写清数据来源、字段、消息、幂等和回滚。
3. 在 Design 写清影响的 service、store、topic 和查询接口。
4. 同步对应后端 SDD，字段和链路变化优先放 `be/specs`。
5. 通过 Gate 后实现。
6. 执行 `make harness-check`、`make test`。
7. 影响主链路跑 `make verify`，影响 CDC 跑 `make verify-debezium`。

### frontend-change

1. 从 `fe/package.json`、`fe/src/`、`fe/vite.config.ts` 和 `fe/tsconfig.json` 修改源码。
2. 不把 `fe/dist` 或 `fe/node_modules` 当源码入口。
3. 写清页面入口、API、空态、错误态和移动端验证。
4. 页面结构、路由、API client 或指标口径变化时同步前端 SDD。
5. 执行 `make frontend-build`，再执行 `make harness-check`。
6. 页面行为变化需要浏览器验证主要路径。

### full-stack-change

1. 复制 [tasks/_template](tasks/_template) 建阶段文档。
2. SPEC 必须写清 API contract、页面入口、错误态、权限和验收标准。
3. Design 同时列出后端落点和前端落点。
4. 后端 SDD 记录接口、字段、存储和数据链路；前端 SDD 记录页面、状态和交互契约。
5. 先保证后端接口可验证，再接前端。
6. 执行 `make harness-check`、`make test`、必要的 `make verify`。
7. 执行 `make frontend-build` 和浏览器验证。

### startup-debug

1. 先 `make status`。
2. 查 `be/logs/startup.log` 和对应 `be/logs/*.stdout.log`。
3. 用 `curl http://127.0.0.1:8080/healthz` 交叉验证 `bi-api`。
4. 必要时 `make down && make up` 串行复现。
5. 修脚本后执行 `make harness-check`，影响本地主链路时执行 `make verify`。

## 复杂任务文档

复杂任务复制模板目录：

```bash
cp -R docs/harness/tasks/_template docs/harness/tasks/YYYYMMDD-<slug>
```

阶段文件：

```text
01-spec.md        需求、范围、非目标、验收标准
02-design.md      技术方案、影响文件、接口和数据契约
03-gate.md        是否放行开发，或打回哪一阶段
04-review.md      代码审查结果
05-validation.md  验证结果、未验证项、交付摘要
```

这些阶段文件不替代 SDD。它们记录一次任务的推进过程；如果任务留下了新的接口、字段、页面结构或运行规则，要把长期结论同步回 `be/openspec`、`be/specs`、`fe/openspec` 或 `fe/specs`。

SPEC 至少写清：

- 要解决什么问题。
- 哪些不做。
- 影响后端、前端、数据库、脚本、文档中的哪几块。
- 验收标准是什么。
- 要跑哪些验证。

## 验证

统一入口：

```bash
make harness-check
```

CI 入口同样使用 `make harness-check`，配置在 `.github/workflows/harness-check.yml`。

| 层级 | 命令 | 什么时候必须跑 |
| --- | --- | --- |
| Harness 静态检查 | `make harness-check` | 每次交付默认必须跑 |
| Go 单测 | `make test` | 改 Go 代码、脚本影响编译或接口时 |
| 前端构建 | `make frontend-build` | 改动影响页面或前端 API 时 |
| 本地主链路 | `make verify` | 改采集、转换、BI、启动脚本、存储或消息链路时 |
| Debezium 链路 | `make verify-debezium` | 改 outbox、Debezium、raw events 或 CDC 配置时 |
| 浏览器验证 | 本地页面主要路径 | 改页面、交互、布局、路由或前后端联调时 |

`make harness-check` 当前会检查：

- Harness 文档入口是否存在。
- `workflow.json` 是否为合法 JSON，且包含阶段定义。
- CI workflow 是否仍然调用 `make harness-check`。
- PR / issue 模板和复杂任务模板是否完整。
- `scripts/verify/verify_harness.sh` 是否可执行且语法正确。
- Go 格式和 `go test ./...`。
- 前端构建；缺少 `node_modules` 时先执行 `npm ci`。
- `fe/dist`、`fe/node_modules`、`be/logs`、`be/run` 等生成物是否未被 git 跟踪。
- Harness Markdown 中的本地相对链接是否仍然存在。
- 本机绝对路径和已知本地 token 是否泄漏到仓库文件。

缺少 Go、npm、docker 这类工具时，不能把任务标成已完整验证。可以记录为：

```text
未完整验证：本机缺少 <tool>，已完成 <已跑命令>。
```

只有文档改动且明确不需要 Go/前端验证时，可以临时执行：

```bash
BE_HARNESS_SKIP_GO=1 make harness-check
```

这只能算 docs-only 检查，不能替代代码验证。

## 交付记录

PM 收口或任务完成时更新 [task-board.md](task-board.md)：

- 状态是否进入 `done`。
- 文档入口在哪里。
- 验证跑了什么。
- 还有哪些未验证或阻塞。

反复出现的问题不要只写在聊天里。能机器判断的沉进 `scripts/verify/verify_harness.sh`，不能机器判断的沉进 [AGENTS.md](../../AGENTS.md)、[dev-map.md](dev-map.md) 或对应 runbook。
