# SDD Guide

这份文档是 Harness 里的 SDD 导航，不存放具体设计正文。具体 SDD 仍然留在离代码最近的位置：

```text
be/openspec   后端轻量 SDD
be/specs      后端 Spec Kit SDD
fe/openspec   前端轻量 SDD
fe/specs      前端 Spec Kit SDD
```

## 定义

Harness 管“怎么推进需求”，SDD 管“为什么这样设计、以后怎么延续”。

```text
需求 -> Harness 路由/阶段/验证 -> 判断是否需要 SDD -> 更新对应 SDD -> task-board 收口
```

不要把 SDD 当成另一套入口流程。所有需求先从 [HARNESS.md](../../HARNESS.md) 和 [playbook.md](playbook.md) 进入；只有留下长期接口、字段、链路、页面或运行规则时，才更新 SDD。

## 放在哪里

| 设计内容 | SDD 位置 | 例子 |
| --- | --- | --- |
| 后端服务边界、运行方式、本地启动 | `../../be/openspec`、`../../be/specs` | `002-mac-local-bootstrap` |
| API、字段、数据链路、worker、存储、消息 | `../../be/openspec`、`../../be/specs` | `001-backend-sdd-baseline`、`003-bi-overview-business-data-foundation` |
| 前端页面结构、路由、API client、状态模型 | `../../fe/openspec`、`../../fe/specs` | `001-react-bi-refactor` |
| 指标展示、用户反馈、阶段性限制 | `../../fe/openspec`、必要时同步 `../../be/specs` | `002-bi-overview-user-feedback` |

## 什么时候更新

| 需求类型 | Harness 动作 | SDD 动作 |
| --- | --- | --- |
| 文案、README、脚本说明 | 更新相关文档，跑 `make harness-check` | 通常不需要 |
| 一次性排查 | 在 `task-board.md` 或 issue 记录结论 | 只有沉淀为长期规则时再补 |
| 后端局部 API 或配置 | 查 `dev-map.md`，按 backend-small-change 执行 | 更新 `be/openspec` |
| 数据链路、字段口径、存储或消息变化 | 建 Harness 任务文档 | 更新 `be/specs` |
| 前端页面、路由、API client 或指标展示 | 按 frontend-change 执行 | 更新 `fe/openspec` 或 `fe/specs` |
| 前后端联动 | SPEC 写清 API contract、页面入口和验收标准 | 后端 SDD 写接口/字段/数据来源，前端 SDD 写页面/交互/展示口径 |

## 更新原则

- 优先复用已有 SDD，不为每个小改动新建目录。
- OpenSpec 适合轻量变更说明，Spec Kit 适合跨模块和长期能力。
- Harness task 记录一次任务的过程和验证；SDD 记录后续开发需要继承的设计。
- task-board 只写 SDD 入口和交付结论，不复制整份设计正文。
- 改前端展示口径时，同时确认后端 API、字段和数据来源是否需要更新。

## 当前主要 SDD

后端：

```text
be/openspec/changes/001-backend-sdd-baseline
be/specs/001-backend-sdd-baseline
be/openspec/changes/002-mac-local-bootstrap
be/specs/002-mac-local-bootstrap
be/specs/003-bi-overview-business-data-foundation
```

前端：

```text
fe/openspec/changes/001-react-bi-refactor
fe/specs/001-react-bi-refactor
fe/openspec/changes/002-bi-overview-user-feedback
```

## 收口检查

交付前确认：

- `docs/harness/tasks/*` 或 `task-board.md` 记录了这次任务怎么推进和验证。
- 对应 SDD 记录了长期接口、字段、链路、页面或运行规则。
- README / dev-map / playbook 没有指向旧路径。
- 已执行 `make harness-check`，并按影响面追加 `make test`、`make frontend-build`、`make verify` 或 `make verify-debezium`。
