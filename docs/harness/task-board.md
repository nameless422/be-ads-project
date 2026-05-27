# Task Board

这张表记录项目级任务入口和交付结论。新需求进来先查这里，避免冲掉旧设计或重复造入口。

状态取值：

```text
inbox -> spec -> design -> gate -> implement -> review -> verify -> done
blocked
```

| ID | 状态 | 标题 | 文档入口 | 当前结论 | 验证 |
| --- | --- | --- | --- | --- | --- |
| HAR-001 | done | 按 Harness 工程化文章适配 be_ads_project | `AGENTS.md`、`HARNESS.md`、`docs/harness/*`、`.github/workflows/harness-check.yml` | 已按根入口、开发手册、代码地图、流程边界、任务记录、自动反馈六类收敛；保留 Codex 用法、固定动作、阶段模板、统一验证脚本和 CI gate | `make harness-check` 通过 |
| OPS-001 | done | 本地启动、状态、验收脚本入口 | `README.md`、`scripts/README.md`、`be/scripts/README.md` | 当前以 `make up/start/status/verify/down` 作为稳定入口 | `make verify` |
| DATA-001 | done | raw / trans / bi 字段来源说明 | `docs/architecture/raw-trans-bi-field-lineage.md` | 字段 lineage 以架构文档为权威入口 | 文档审查 |
| FE-001 | done | 恢复可维护的前端源码入口 | `fe/package.json`、`fe/src/`、`fe/README.md` | 已从历史分支恢复 React/Vite/TypeScript 前端源码，生成物仍保持忽略 | `make harness-check` 会执行前端构建 |

## 更新规则

- PM 收口时更新状态、文档入口和交付结论。
- 新增复杂任务时创建 `docs/harness/tasks/YYYYMMDD-<slug>/`，并把入口写回本表。
- 验证失败但任务需要继续推进时，状态保持 `blocked` 或 `verify`，不能写成 `done`。
