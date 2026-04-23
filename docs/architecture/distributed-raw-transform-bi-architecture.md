# 广告投放系统目标架构 V2

这版架构以 3 个核心目标为主：

1. 数据先落 `raw db`，采集与转换彻底解耦。
2. 收集层和转换层都可以独立横向扩容。
3. 监控、追踪、日志、部署形态和存储选型都可替换。

## 一、目标分层

最终建议拆成 4 类服务，而不是继续把所有职责放在一个进程里：

- `control-plane`
- `collector-worker`
- `transformer-worker`
- `bi-api`

配套基础设施：

- `raw mysql`
- `message bus`
- `serving mysql`
- `clickhouse`
- `otel collector`
- `prometheus + grafana + loki/tempo`

## 二、核心数据流

```text
control-plane
  -> dispatch collect jobs
  -> collector-worker
  -> raw mysql (raw_records / checkpoints / outbox)
  -> CDC / message bus
  -> transformer-worker
  -> serving mysql + clickhouse + other sinks
  -> bi-api
```

### 1. 数据收集层

职责：

- 接收中心服务分发的采集任务
- 按账号/平台/API 配置拉取原始数据
- 只做轻量标准包裹，不做业务字段标准化
- 把数据写入 `raw db`
- 记录 checkpoint / cursor / watermark
- 发出 `raw_ingested` 事件

建议 raw 表结构至少包含：

- `job_id`
- `platform`
- `platform_account_id`
- `object_type`
- `resource_id`
- `payload_json`
- `source_cursor`
- `source_watermark`
- `collected_at`
- `trace_id`

### 2. 数据转换层

职责：

- 监听 `raw_ingested` 事件
- 拉取 raw 数据
- 做字段标准化、实体拆分、指标换算、币种/时间标准化
- 把转换结果投影到多个下游

下游至少分两类：

- `serving mysql`
  - 适合账号、campaign、ad_group、ad 这类 OLTP 查询
- `clickhouse`
  - 适合 insight、聚合分析、报表、趋势查询

### 3. BI 查询层

职责：

- 不直接依赖采集逻辑
- 只依赖 serving mysql / clickhouse 的读模型
- 提供筛选、分页、聚合、趋势、维度下钻

### 4. 控制层

职责：

- 统一管理采集任务
- 做调度、租约、失败重试、限流、熔断、优先级控制
- 统一管理 worker 注册与心跳

## 三、推荐的职责边界

### control-plane

建议负责：

- profile 管理
- job 生成
- shard 分配
- worker lease
- retry / dead-letter
- rate limit
- job dashboard

不建议负责：

- 真正调用平台 API
- 真正做数据转换
- 直接承担 BI 查询

### collector-worker

建议负责：

- connector 执行
- 幂等写 raw
- checkpoint 更新
- 上报 job 状态

不建议负责：

- 写 serving mysql
- 写 clickhouse
- 复杂字段标准化

### transformer-worker

建议负责：

- raw -> normalized
- normalized -> projection
- 多 sink 投影
- 异常隔离和重放

### bi-api

建议负责：

- 只读接口
- query model
- 聚合缓存
- 限流与权限

## 四、扩容模型

## 4.1 收集层扩容

收集层建议做成“中心调度 + 多 worker 拉取”的模式。

更贴近你要的 map-reduce 思路：

- `map`
  - 每个 worker 拉取一个或一批 platform/account/object 任务
- `reduce`
  - 不是在 collector 做 reduce
  - 而是在 transformation / bi query 做聚合投影

推荐策略：

- control-plane 负责切 job
- job 粒度建议是：
  - `platform + account + object_type + time_range`
- collector-worker 只抢自己能处理的 job
- 每个 job 带 lease timeout
- 超时未 ack 自动回收再分配

## 4.2 转换层扩容

转换层不建议主动扫 raw 表全表轮询。

推荐两种方式：

1. `Outbox + CDC`
2. `直接 message bus`

更稳妥的推荐是：

- collector 把 raw 和 outbox 事件写在同一事务里
- Debezium 把 outbox 事件推到总线
- transformer-worker 订阅事件再做转换

这样好处是：

- 不丢事件
- 采集成功和事件发布有事务一致性
- 转换层天然异步
- 后面增加新 sink 不影响采集层

## 五、存储职责建议

### raw mysql

用途：

- 原始数据落地
- checkpoint
- job 状态
- outbox 事件

特点：

- OLTP
- 强事务
- 便于管理 job、租约、幂等键

### serving mysql

用途：

- campaign / ad_group / ad / account 的查询
- BI API 的维度筛选
- 后台运营页面

特点：

- 行级更新多
- 主键 upsert 清晰
- 适合作为“服务型查询库”

### clickhouse

用途：

- insight
- 趋势聚合
- 多维报表
- 大时间范围查询

特点：

- 适合 append / batch upsert 风格的分析数据
- 高并发聚合性能好

## 六、消息系统推荐

### 推荐优先级

1. `NATS JetStream`
2. `Kafka / Redpanda`

### 推荐 NATS JetStream 的原因

更适合这个项目当前阶段：

- 部署更轻
- Go 生态简单
- pull consumer 容易做 worker 横向扩容
- at-least-once 语义清晰
- 很适合 job dispatch 和 transformer 订阅

适合承接两个通道：

- `collect.jobs`
- `raw.events`

## 七、工作流与调度推荐

### 推荐组合

- 普通采集任务调度：`control-plane + JetStream`
- 长事务/复杂重试编排：`Temporal`

建议不要一上来把所有东西都做成 Temporal workflow。

更合适的边界是：

- 高频、短任务：消息队列驱动
- 跨多阶段、强可追踪、需要人工介入的任务：Temporal

比如这些适合 Temporal：

- 首次大账号 bootstrap
- 手动重放某批 raw 数据
- 某账号授权失效后的恢复流程
- 需要多步补偿的重算任务

## 八、观测性设计

建议从第一天就统一三件事：

- logs
- metrics
- traces

### logs

要求：

- JSON structured logs
- 每条日志带：
  - `trace_id`
  - `job_id`
  - `platform`
  - `platform_account_id`
  - `object_type`
  - `worker_id`

### metrics

关键指标：

- job dispatch count
- job success / failure count
- connector latency
- raw write latency
- transform latency
- sink write latency
- queue lag
- raw backlog
- retry count
- dead letter count

### traces

一条完整 trace 建议跨这些阶段：

- control-plane dispatch
- collector fetch
- raw db write
- outbox publish / CDC emit
- transformer consume
- mysql projection
- clickhouse projection
- bi query

## 九、代码结构建议

当前仓库建议逐步改造成下面这种结构：

```text
cmd/
  control-plane/
  collector-worker/
  transformer-worker/
  bi-api/

internal/
  collection/
    domain/
    application/
    infrastructure/
  rawstore/
    domain/
    application/
    infrastructure/
  transformation/
    domain/
    application/
    infrastructure/
  serving/
    domain/
    application/
    infrastructure/
  biquery/
    domain/
    application/
    infrastructure/
  scheduling/
    domain/
    application/
    infrastructure/
  messaging/
    domain/
    infrastructure/
  shared/
    observability/
    config/
    middleware/
    platform/
```

关键原则：

- `domain`
  - 只放业务模型、领域规则、端口
- `application`
  - 只放用例编排
- `infrastructure`
  - 只放 DB、MQ、HTTP、SDK、CDC、具体实现

## 十、推荐开源组件

### 微服务框架

推荐：

- `Kratos`

理由：

- Go 微服务治理能力完整
- HTTP / gRPC / middleware / tracing / metrics 都成熟
- 对后续拆分 `control-plane / bi-api / worker admin api` 很友好

### 消息系统

推荐：

- `NATS JetStream`

### CDC / Outbox

推荐：

- `Debezium`

### 可观测性

推荐：

- `OpenTelemetry`
- `Prometheus`
- `Grafana`
- `Loki`
- `Tempo` 或 `Jaeger`

### 容器编排

开发环境：

- `docker compose`

生产环境：

- `Kubernetes`

## 十一、部署建议

### 单机开发

适合：

- control-plane
- collector-worker
- transformer-worker
- bi-api
- mysql
- clickhouse
- nats
- otel-collector

### 小规模生产

建议：

- control-plane 2 副本
- collector-worker 多副本
- transformer-worker 多副本
- bi-api 2 副本
- mysql 主从或云 RDS
- clickhouse 集群或云服务
- nats 3 节点

## 十二、迁移路线

### Phase 1

把当前单进程拆成：

- `collector-worker`
- `transformer-worker`
- `bi-api`

先保留本地消息总线实现。

### Phase 2

引入：

- raw mysql
- serving mysql
- clickhouse
- NATS JetStream

### Phase 3

引入：

- outbox
- Debezium CDC
- 死信队列
- replay 工具

### Phase 4

引入：

- Kratos 化服务治理
- OTel 全链路追踪
- Prometheus / Grafana / Loki / Tempo
- K8S 部署

## 十三、最终建议

如果按“稳定、易维护、易扩展”来权衡，建议你的目标技术栈是：

- `control-plane / bi-api / workers`: Go + Kratos
- `raw / serving`: MySQL
- `olap`: ClickHouse
- `message bus`: NATS JetStream
- `cdc/outbox`: Debezium
- `observability`: OpenTelemetry + Prometheus + Grafana + Loki + Tempo

这是一个比较稳的组合：

- 采集层和转换层都能横向扩
- 原始数据和服务数据分离
- BI 查询不污染采集链路
- 底层数据库和消息系统后续可替换
- 监控、日志、追踪和部署边界都清楚
