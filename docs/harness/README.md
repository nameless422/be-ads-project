# Harness 总览

这套 Harness 按文章里的四块拼图落到本仓库：

| 分类 | 本仓库资产 | 管什么 |
| --- | --- | --- |
| 根入口 | `AGENTS.md`、`HARNESS.md` | AI 和人先读什么、不能碰什么、交付定义是什么 |
| 开发手册 | `playbook.md` | Codex 用法、任务类型、固定动作、验证矩阵、交付记录 |
| 代码地图 | `dev-map.md` | 后端、前端、脚本、文档分别改哪里 |
| 流程边界 | `workflow.md`、`workflow.json` | 阶段流转、角色边界、回退规则、机器可读状态表 |
| 任务记录 | `task-board.md`、`tasks/_template/` | 当前任务状态和复杂任务阶段文档 |
| 自动反馈 | `scripts/verify/verify_harness.sh`、`.github/workflows/harness-check.yml` | 交付结果由脚本和 CI 判定，不只靠口头说明 |

## 仓库画像

`be_ads_project` 是广告数据平台，主链路是：

```text
control-plane
  -> collector-worker
  -> raw mysql / outbox
  -> JetStream / Debezium
  -> transformer-worker
  -> serving mysql + clickhouse
  -> bi-api
```

后端源码、Go module、vendor、运行脚本和后端配置都在 `be/`。前端源码在 `fe/package.json` 和 `fe/src/`，前端生成物 `fe/dist` 与 `fe/node_modules` 不作为源码权威。根目录 `scripts/` 只保留跨前后端的 Harness 检查。

## 使用流程

1. AI 执行者先读根规则 [../../AGENTS.md](../../AGENTS.md)。
2. 新需求先查 [task-board.md](task-board.md)，确认是不是旧任务延续。
3. 动代码前查 [dev-map.md](dev-map.md)，确认模块落点和影响面。
4. 需求进入开发时按 [playbook.md](playbook.md) 判断轻重、Codex 用法和验证范围。
5. 复杂任务先写 SPEC，模板见 [tasks/_template/01-spec.md](tasks/_template/01-spec.md)。
6. 按 [workflow.md](workflow.md) 走阶段；机器可读定义见 [workflow.json](workflow.json)。
7. 收尾默认先跑 `make harness-check`。

## 维护规则

- 改了目录或服务边界：更新 `dev-map.md` 和 `docs/README.md`。
- 改了运行或验证方式：更新 `playbook.md`、`scripts/README.md`、`be/scripts/README.md` 和 `Makefile help`。
- 改了任务阶段、结论或阻塞项：更新 `task-board.md`。
- 能被脚本判定的规则，优先沉到 `scripts/verify/verify_harness.sh`，再在文档里解释。
- 复杂任务复制 `docs/harness/tasks/_template/`，不要把阶段结论只留在聊天里。
- PR 走 `.github/PULL_REQUEST_TEMPLATE.md`，新需求优先用 `.github/ISSUE_TEMPLATE/feature_request.yml` 收集目标、范围和验收标准。
