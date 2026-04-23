# Implementation Roadmap

这份路线图把当前技术方案拆成可以连续推进的阶段，目标是把系统逐步落到：

- `control-plane`
- `collector-worker`
- `transformer-worker`
- `bi-api`

并最终收敛到：

- `Go + Kratos`
- `MySQL(raw / serving)`
- `ClickHouse`
- `NATS JetStream`
- `Debezium Outbox / CDC`
- `OpenTelemetry`

## 当前状态

### 已完成

- `control-plane -> collector-worker -> transformer-worker -> bi-api` 4 个入口已经拆开
- `collector-worker` 已通过 `JetStream pull consumer` 拉取采集任务
- 采集结果已先落 `raw mysql.raw_records`
- `transformer-worker` 已通过 `JetStream pull consumer` 消费 `raw.events`
- 标准化结果已写入：
  - `serving mysql`
  - `clickhouse`
- `bi-api` 已能查询：
  - `snapshots`
  - `campaigns`
  - `insights summary`
- 本地 Phase 1 基础设施已可通过纯 `docker run` 启动
- 已有自动化验收脚本：
  - [verify_phase1.sh](/Users/zhongyi.zhang/project/go/be_ads_project/scripts/verify/verify_phase1.sh)

### 当前边界

- `raw_records + outbox_events` 已进入同事务写入
- `collector-worker` 已增加 outbox relay，负责把 pending event 投递到 JetStream 并回写状态
- `Debezium CDC` 已接入并完成本地联调
- 4 个入口服务已切到 `Kratos` runtime
- `OTel tracing/metrics/logs` 还没接入代码
- 任务分发和 worker 租约仍是轻量实现

## Phase 1

目标：先把异步链路和分层边界跑通。

### 验收标准

- control-plane 能周期性下发 job
- collector-worker 能消费 job 并写 raw mysql
- transformer-worker 能消费 raw event 并写 serving mysql / clickhouse
- bi-api 能查到 MySQL / ClickHouse 数据

### 当前结果

这一阶段已经完成。

## Phase 2

目标：把 `raw 数据入库` 和 `raw 事件投递` 变成更可靠的一致性链路。

### 任务拆解

1. 新增 `raw_mysql.outbox_events`
2. collector 在同一事务里写：
   - `raw_records`
   - `outbox_events`
3. 增加 outbox relay，定期扫描 pending/failed 事件并投递
4. 为 outbox 增加重试和退避
5. 为 outbox 增加重放和清理策略

### 完成标准

- collector 不再依赖“先写库再单独发消息”的非原子流程
- outbox 表支持重试、重放、审计

### 当前结果

这一阶段已经完成了第一版：

- `outbox_events` 已落库
- `raw_records + outbox_events` 已在同一事务提交
- `collector-worker` 已通过 relay 异步发布 pending event
- 发布结果会回写：
  - `published`
  - `failed`

下一步就是把 relay 从应用内轮询升级为 `Debezium CDC`。

## Phase 3

目标：引入 `Debezium CDC`，把 outbox 事件正式解耦为 CDC 流。

### 任务拆解

1. 增加 `debezium server` 本地运行配置
2. 监听 `raw_mysql.outbox_events`
3. 将 outbox 事件转发到：
   - `JetStream`
   - 或兼容的下游总线主题
4. transformer-worker 改成只消费 CDC 后的标准事件
5. 增加 lag / connector health / retry 观测

### 完成标准

- collector 不直接发布 raw 事件
- raw db 和事件总线之间的同步依赖 CDC

### 当前结果

这一阶段已经完成了第一版本地接入：

- 已新增 Debezium 本地启动脚本
- 已新增 `BE_OUTBOX_TRANSPORT=debezium` 运行模式
- collector 在该模式下不会启用应用内 relay
- Debezium 会从 `raw_mysql.outbox_events` 读取变更并发送到 `raw.events.ingested`

当前仍未完成的部分是：

- Debezium metrics / tracing
- outbox 清理策略
- 多环境部署配置

## Phase 4

目标：把服务治理切到 `Kratos`。

### 任务拆解

1. 为 4 个入口服务建立 Kratos app skeleton
2. 统一：
   - config
   - logger
   - metrics
   - middleware
   - health/readiness
3. 把 HTTP 和 worker lifecycle 接到 Kratos runtime
4. 为控制面和 worker 预留 gRPC 能力

### 完成标准

- 4 个服务统一生命周期和配置模型
- 为后面多环境部署做好准备

### 当前结果

这一阶段已经完成。

已落地内容：

- `cmd/control-plane`
- `cmd/collector-worker`
- `cmd/transformer-worker`
- `cmd/bi-api`

都已经统一挂到 `Kratos` app runtime。

其中：

- `bi-api` 已切到 `Kratos HTTP server`
- worker 已切到统一的 `Kratos` lifecycle server wrapper
- 旧的单体入口和过时同步链路代码已移除
- 本地验证已覆盖：
  - `go test ./...`
  - `GET /healthz`
  - `GET /api/bi/snapshots`
  - control-plane / collector-worker / transformer-worker 启停与日志

## Phase 5

目标：接入 `OpenTelemetry` 和生产级观测。

### 任务拆解

1. 打通 trace：
   - control-plane dispatch
   - collector fetch
   - raw store
   - transform
   - projector
   - bi query
2. 暴露 metrics：
   - job dispatch count
   - consumer lag
   - raw write latency
   - transform latency
   - sink failure count
3. 统一结构化日志字段：
   - trace_id
   - job_id
   - event_id
   - profile_id
   - platform
   - account_id

### 完成标准

- 一条 job 能跨 4 层追踪
- Prometheus/Grafana 可观测关键指标

## Phase 6

目标：提升扩展性和生产稳定性。

### 任务拆解

1. 引入 worker lease / shard / backpressure
2. 支持 platform 维度 consumer 拆分
3. 支持 replay / backfill / dead letter
4. 增加原始 payload 保留策略和归档策略
5. 补齐部署清单：
   - local
   - dev
   - staging
   - prod

### 完成标准

- collector / transformer 都可按机器数横向扩展
- 大批量补数不影响主同步链路

## 推荐执行顺序

1. `Phase 2 Outbox`
2. `Phase 3 Debezium`
3. `Phase 5 OTel`
4. `Phase 4 Kratos`
5. `Phase 6 扩展性治理`

这个顺序的原因是：

- 先把数据一致性补稳
- 再把异步链路从应用直发升级到 CDC
- 接着补可观测，方便后续重构
- 最后再做 Kratos 化和生产级扩容治理
