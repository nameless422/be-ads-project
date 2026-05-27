# be_ads_project 总文档

这份文档是项目入口，只负责三件事：

- 说明项目做什么
- 说明业务链路是什么
- 给出技术架构总览

如果你需要更细的内容，按下面分工继续看：

- [Harness Entry](../HARNESS.md)
  只看 AI/人协作流程、dev-map、任务看板和统一验证入口
- [ADR-001 Target Stack](architecture/adr-001-target-stack.md)
  只看技术选型决策和为什么这样选
- [Distributed Raw Transform BI Architecture](architecture/distributed-raw-transform-bi-architecture.md)
  只看服务边界、数据流和扩容模型
- [Raw Trans BI Field Lineage](architecture/raw-trans-bi-field-lineage.md)
  只看 raw/trans/bi 当前实际字段、来源和取数逻辑
- [Google Ads Test To Real Runbook](runbooks/google-ads-test-to-real-runbook.md)
  只看 Google Ads 接入和从测试切到真实账号的操作

## 项目介绍

`be_ads_project` 是一个广告投放数据平台项目，目标是把不同广告平台的数据采集、转换、存储和 BI 查询拆成清晰可扩展的链路，而不是继续堆在单个进程里。

当前项目已经落成 4 个核心服务：

- `control-plane`
- `collector-worker`
- `transformer-worker`
- `bi-api`

仓库按前后端分区维护：

- `be/`
  后端 Go module、服务入口、internal 模块、vendor、后端脚本和后端配置。
- `fe/`
  React / Vite / TypeScript 前端源码。
- `scripts/`
  跨前后端的仓库级 Harness 检查脚本。
- `docs/`
  架构、runbook 和 Harness 文档入口。

当前主链路已经支持：

- 任务分发与 worker lease / shard 分配
- 平台数据采集并落 raw mysql
- raw 事件异步传播
- 标准化结果投影到 serving mysql / clickhouse
- BI 查询接口

## 业务介绍

这个项目解决的是“多广告平台数据如何稳定进入内部分析链路”的问题。

业务流程分成 4 段：

1. `任务生成`
   control-plane 按平台、账号、对象类型和时间范围生成采集任务。
2. `原始采集`
   collector-worker 调用平台 API，把结果按原始形态落库，并记录 checkpoint、outbox 和执行状态。
3. `标准化转换`
   transformer-worker 消费 raw 事件，把不同平台字段映射成统一模型，并投影到查询库和分析库。
4. `BI 查询`
   bi-api 只依赖读模型，对外提供 snapshot、campaign、insight summary 等查询接口。

当前业务对象主要包括：

- account
- campaign
- ad_group
- ad
- insight

当前平台接入以这些为主：

- Google Ads
- Facebook
- TikTok Ads

Google Ads 当前支持两种模式：

- `seeded_test`
  用仓库内置测试账号和本地生成 payload 先验证链路
- `real`
  用真实 OAuth 凭证直接调用 Google Ads API

## 技术架构

### 服务拆分

- `control-plane`
  负责 job 生成、worker lease、shard 分配、失败重试和调度治理
- `collector-worker`
  负责 connector 执行、raw 入库、outbox 写入和采集侧状态维护
- `transformer-worker`
  负责 raw 标准化、多 sink 投影、异常隔离和重放处理
- `bi-api`
  负责只读查询接口，不直接参与采集和转换

### 存储与消息链路

- `raw mysql`
  存储 raw_records、outbox_events、checkpoint、worker_leases、shard_assignments 等强事务数据
- `serving mysql`
  存储 account、campaign、ad_group、ad 等维度模型
- `clickhouse`
  存储 insight、趋势、聚合分析等 OLAP 数据
- `NATS JetStream`
  承载 collect.jobs 和 raw.events 等异步消息
- `Debezium`
  监听 raw mysql 的 outbox_events，把事件稳定转发到消息总线

### 核心数据流

```text
control-plane
  -> collect.jobs.<platform>.shard.<id>
  -> collector-worker
  -> raw mysql(raw_records / outbox_events / checkpoints)
  -> Debezium / JetStream raw.events.<platform>
  -> transformer-worker
  -> serving mysql + clickhouse
  -> bi-api
```

### 当前技术选型

- `Go`
- `Kratos`
- `MySQL`
- `ClickHouse`
- `NATS JetStream`
- `Debezium Outbox / CDC`
- `OpenTelemetry`
