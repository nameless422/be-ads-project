# be_ads_project

当前仓库已初始化为广告投放系统的三层骨架，主链路已经按：

- `数据收集层`
- `数据转换层`
- `BI 查询层`

进行拆分，方便后续继续接真实存储和查询接口。

## 当前包含

- `internal/modules/collection` 数据收集业务域
- `internal/modules/transformation` 数据转换业务域
- `internal/modules/reporting` BI 查询业务域
- `internal/modules/controlplane` 任务分发业务域
- `cmd/control-plane` 任务分发入口
- `cmd/collector-worker` 数据采集 worker
- `cmd/transformer-worker` raw 转换 worker
- `cmd/bi-api` BI 查询 API
- 4 个服务统一接入 `Kratos` 生命周期
- 平台 connector 抽象
- Facebook / Google Ads / TikTok 接入骨架
- 标准化领域模型
- mock 账号与 seeded test account 拉取数据
- 本地持续日志输出
- MySQL OLTP 写入
- ClickHouse OLAP 写入
- BI HTTP 查询接口
- raw outbox + Debezium CDC 链路

## 推荐目录结构

当前仓库建议按下面的方式理解：

```text
cmd/
  control-plane/         中心调度与 job 分发
  collector-worker/      拉平台数据，写 raw + outbox
  transformer-worker/    消费 raw event，做标准化并投影
  bi-api/                BI 查询接口
  google_oauth_bootstrap/ Google Ads OAuth 辅助工具

internal/
  modules/
    controlplane/
      application/       调度用例编排
      domain/            job 构建与调度端口
      infrastructure/    JetStream 等实现
    collection/
      application/       采集执行与 outbox relay
      domain/            sync target / collected batch / raw outbox 模型
      infrastructure/
        collector/       采集编排
        provider/        多账户 profile 提供者
        connectors/      Facebook / Google / TikTok connector
        jetstream/       采集事件发布
        rawstore/mysql/  raw_records / outbox_events 持久化
    transformation/
      application/       转换编排与 worker 事件处理
      domain/            normalized batch / projector / normalizer 端口
      infrastructure/
        normalizer/      平台标准化映射
        projector/       fanout 与多库投影
    reporting/
      application/       BI snapshot 投影逻辑
      domain/            查询视图与 repository 端口
      infrastructure/
        httpapi/         BI HTTP 接口
        mysql/           OLTP 查询实现
        clickhouse/      OLAP 查询实现
  shared/
    ads/                 跨业务域共享广告核心模型
    messaging/           跨业务域共享消息模型与 JetStream client
  platform/              mysql / clickhouse / kratosx 等基础设施适配
  config/                环境配置
  logx/                  日志初始化
  mock/                  mock 账号与 seeded test 数据

scripts/
  dev/
    dev_phase1_stack_*.sh    本地 MySQL / ClickHouse / NATS 基础设施
    dev_debezium_*.sh        Debezium 本地联调
    dev_stack_*.sh           docker compose 辅助脚本
  ops/
    start.sh                 启动 4 个服务
    stop.sh                  停止 4 个服务
    status.sh                查看状态与最近日志
  verify/
    verify_phase1.sh         Phase 1 验收
    verify_phase3_debezium.sh Phase 3 验收

docs/
  architecture/              当前生效的架构与 ADR
  runbooks/                  平台接入和联调手册
  roadmap/                   阶段路线图
  archive/                   历史设计草案
```

现在已经清掉的旧残留包括：

- 旧单体入口 `cmd/server`
- 旧散落式业务/技术混合目录
- 旧内存版 BI repository
- 旧单库脚本 `scripts/dev_db_up.sh` / `scripts/dev_db_down.sh`

## 当前不包含

- 数据库存储实现
- 完整 BI API
- 完整任务队列与重试实现
- 默认不强依赖真实平台 API 调用

## 本地运行

### Phase 1 拆分服务

先启动基础设施：

```bash
./scripts/dev/dev_phase1_stack_up.sh
```

再启动 4 个服务：

```bash
./scripts/ops/start.sh
```

查看状态：

```bash
./scripts/ops/status.sh
```

自动验收：

```bash
./scripts/verify/verify_phase1.sh
```

### Phase 4 Kratos

当前 4 个入口服务都已经切到 `Kratos` app runtime：

- `cmd/control-plane`
- `cmd/collector-worker`
- `cmd/transformer-worker`
- `cmd/bi-api`

`bi-api` 额外接了：

- HTTP access log
- panic recovery
- 统一优雅停机

### Phase 3 Debezium

如果要验证 `outbox -> Debezium -> JetStream`：

```bash
./scripts/dev/dev_phase1_stack_down.sh
./scripts/dev/dev_phase1_stack_up.sh
./scripts/dev/dev_debezium_up.sh
BE_OUTBOX_TRANSPORT=debezium ./scripts/ops/start.sh
./scripts/verify/verify_phase3_debezium.sh
```

如果某个 Debezium 镜像版本在本机上不稳定，可以覆盖：

```bash
DEBEZIUM_IMAGE=quay.io/debezium/server:3.4.3.Final ./scripts/dev/dev_debezium_up.sh
```

关闭 Debezium：

```bash
./scripts/dev/dev_debezium_down.sh
```

停止服务：

```bash
./scripts/ops/stop.sh
```

停止基础设施：

```bash
./scripts/dev/dev_phase1_stack_down.sh
```

日志位于：

- `logs/control-plane.log`
- `logs/collector-worker.log`
- `logs/transformer-worker.log`
- `logs/bi-api.log`
- `logs/*.stdout.log`

## BI 接口

`bi-api` 启动后默认监听 `:8080`，可直接访问：

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/bi/snapshots
curl 'http://127.0.0.1:8080/api/bi/campaigns?platform=tiktok_ads&account_id=tt-act-3001'
curl 'http://127.0.0.1:8080/api/bi/insights/summary?platform=tiktok_ads&platform_account_id=acct_tt_001&date_from=2026-04-22&date_to=2026-04-22'
```

## Google Ads Test Accounts

当前已经内置 5 个 Google Ads test customer，用于本地联调整条采集链路：

- `248-390-1805`
- `492-825-2952`
- `691-332-4649`
- `400-404-3492`
- `608-174-8445`

默认运行模式是 `seeded_test`，会按这些 test account 的真实 `customer_id` 生成可观测的测试 payload 并打印 raw/normalized 日志。

如果你后面已经拿到 Google Ads OAuth 凭证，可以通过环境变量切到真实 API：

```bash
export BE_GOOGLE_ADS_MODE=real
export BE_GOOGLE_ADS_DEVELOPER_TOKEN=your_developer_token
export BE_GOOGLE_ADS_LOGIN_CUSTOMER_ID=357-594-0005
export BE_GOOGLE_ADS_CLIENT_ID=your_client_id
export BE_GOOGLE_ADS_CLIENT_SECRET=your_client_secret
export BE_GOOGLE_ADS_REFRESH_TOKEN=your_refresh_token
```

完整执行说明见：

- [Google Ads Test To Real Runbook](/Users/zhongyi.zhang/project/go/be_ads_project/docs/runbooks/google-ads-test-to-real-runbook.md)
- [Distributed Raw Transform BI Architecture](/Users/zhongyi.zhang/project/go/be_ads_project/docs/architecture/distributed-raw-transform-bi-architecture.md)
- [ADR-001 Target Stack](/Users/zhongyi.zhang/project/go/be_ads_project/docs/architecture/adr-001-target-stack.md)
- [Implementation Roadmap](/Users/zhongyi.zhang/project/go/be_ads_project/docs/roadmap/implementation-roadmap.md)

## 目标技术栈

当前目标技术栈已经确认：

- `Go + Kratos`
- `MySQL(raw/serving)`
- `ClickHouse`
- `NATS JetStream`
- `Debezium Outbox / CDC`
- `OpenTelemetry`

对应决策文档：

- [ADR-001 Target Stack](/Users/zhongyi.zhang/project/go/be_ads_project/docs/architecture/adr-001-target-stack.md)

## 本地基础设施

启动本地基础设施：

```bash
./scripts/dev/dev_stack_up.sh
```

关闭本地基础设施：

```bash
./scripts/dev/dev_stack_down.sh
```

本地 `docker compose` 默认会拉起：

- `raw-mysql`
- `serving-mysql`
- `clickhouse`
- `nats` with `JetStream`
- `otel-collector`

对应配置模板：

- [.env.stack.example](/Users/zhongyi.zhang/project/go/be_ads_project/.env.stack.example)

## 推荐下一步

1. 进入 `Phase 5 OpenTelemetry`，把 trace / metrics / logs 串起来。
2. 为 control-plane 和 worker 增加更清晰的 job lease / retry / DLQ 监控。
3. 为 Facebook / Google / TikTok 补真实 connector 的生产鉴权与限流治理。
