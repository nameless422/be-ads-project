# ADR-001 目标技术栈确认

状态：

- accepted

日期：

- 2026-04-23

## 决策

广告投放系统后续目标技术栈确定为：

- `Go`
- `Kratos`
- `MySQL`
  - `raw mysql`
  - `serving mysql`
- `ClickHouse`
- `NATS JetStream`
- `Debezium Outbox / CDC`
- `OpenTelemetry`

## 背景

系统目标已经从单进程采集演进为分布式数据平台，核心要求包括：

- 采集层与转换层解耦
- raw 数据先落库
- 转换层异步消费 raw 事件
- 多个下游存储可扩展
- BI 查询层只依赖读模型
- collector / transformer 可以横向扩容
- 具备较清晰的日志、trace、metrics 和部署边界

## 组件职责

### Go + Kratos

用于承载以下服务：

- `control-plane`
- `collector-worker`
- `transformer-worker`
- `bi-api`

选择原因：

- 适合 Go 微服务拆分
- 内置中间件、配置、日志、metrics、tracing 生态较完整
- 后续治理能力比纯裸 `net/http` 更稳定

### raw mysql

存储：

- raw records
- checkpoints
- leases
- jobs
- outbox events

选择原因：

- 强事务
- 适合任务、租约、游标、幂等控制
- 便于与 outbox 模式结合

### serving mysql

存储：

- account
- campaign
- ad_group
- ad
- BI 查询维度表

选择原因：

- 适合面向服务的读写模型
- 适合后台筛选和管理接口

### ClickHouse

存储：

- insight
- 指标汇总
- 趋势报表
- 多维聚合查询

选择原因：

- 适合 OLAP 聚合
- 时间范围大时性能更稳定

### NATS JetStream

承载：

- `collect.jobs.*`
- `raw.events.*`
- 后续可扩展 `replay.jobs.*`

选择原因：

- pull consumer 适合 worker 横向扩容
- 运维复杂度比 Kafka 更低
- 比较适合当前阶段的任务分发和事件总线

### Debezium Outbox / CDC

承载：

- `raw mysql.outbox_events` -> message bus

选择原因：

- 保证 raw 写入和事件发布的一致性
- 让 transformer 与 collector 强解耦
- 方便后续增加更多消费者

### OpenTelemetry

承载：

- trace
- metric
- log correlation

选择原因：

- 标准化程度高
- 后续可接 Prometheus / Grafana / Tempo / Loki

## 初期实施顺序

### Phase 1

- 固化技术栈和目录边界
- 本地基础设施统一成 `docker compose`
- 引入 `NATS JetStream`
- 拆出 `control-plane / collector-worker / transformer-worker / bi-api`

### Phase 2

- collector 只写 `raw mysql`
- transformer 消费 `raw.events`
- serving mysql / clickhouse 投影稳定

### Phase 3

- collector 写 `outbox_events`
- Debezium 监听 outbox 并转发消息

### Phase 4

- 全链路 OTel
- K8S 部署
- 弹性扩容与死信重放

## 不选其他主方案的原因

### Kafka 作为当前阶段主总线

不选原因：

- 更重
- 运维复杂度更高
- 当前阶段收益不如 JetStream 明显

### Temporal 作为主任务骨架

不选原因：

- 更适合长流程编排
- 不适合替代高频任务队列和事件总线

### Airflow / Argo 作为主链路

不选原因：

- 更偏 DAG 调度
- 不适合高频实时 worker 消费主链路
