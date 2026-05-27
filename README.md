# be-ads-project

面向广告投放数据的本地采集、转换、分析和 BI 展示系统。

系统按广告平台账号组织采集任务，写入 raw 数据，经过标准化转换后投影到查询存储，最后通过 BI API 和 React 页面提供报表分析能力。项目同时提供一套本地运维控制面，方便在开发环境启动、停止、查看和验收整条链路。

## 核心能力

- 多平台广告数据采集骨架：Google Ads、Facebook、TikTok
- raw 数据落库和 outbox 事件发布
- raw event 标准化转换
- MySQL / ClickHouse 查询投影
- BI 查询接口：账号、Campaign、Insight、Search Term、UA、素材质量等
- React BI 页面：Overview、Breakdown、Creatives、Quality、Control
- 本地控制面：启动、停止、状态查看、worker 控制、DLQ 可视化
- Mac 本地新环境一键启动和验收

## 技术选型

| 层 | 技术 | 用途 |
| --- | --- | --- |
| 后端语言 | Go | 服务入口、采集、转换、BI API |
| 后端框架 | Kratos | HTTP 服务生命周期、日志、恢复和访问日志 |
| 前端 | React + Vite + TypeScript | BI 页面和本地控制面 |
| OLTP 存储 | MySQL | raw 数据、outbox、BI 快照和控制数据 |
| OLAP 存储 | ClickHouse | Insight / UA 等分析查询 |
| 消息系统 | NATS JetStream | 采集事件和转换事件 |
| CDC | Debezium | outbox 到消息流的可选联调路径 |
| 本地环境 | Docker Desktop | MySQL、ClickHouse、NATS 等依赖 |
| 开发入口 | Makefile + shell scripts | 本地启动、停止、验证 |
| Harness | AGENTS + HARNESS + docs/harness | AI/人协作流程、任务阶段、验证 gate |
| SDD | OpenSpec + Spec Kit | 需求、方案、任务和长期约束 |

## 系统架构

```mermaid
flowchart LR
  subgraph Platforms["广告平台"]
    Google["Google Ads"]
    Facebook["Facebook"]
    TikTok["TikTok"]
  end

  subgraph Backend["Go 后端"]
    Control["control-plane\n任务分发"]
    Collector["collector-worker\n采集 worker"]
    RawStore["raw MySQL\nraw_records / outbox_events"]
    NATS["NATS JetStream\n事件流"]
    Transformer["transformer-worker\n标准化转换"]
    Serving["serving MySQL\nBI 快照 / 控制数据"]
    ClickHouse["ClickHouse\n分析明细"]
    API["bi-api\n/api/bi/*\n/api/control/*"]
  end

  subgraph Frontend["React 前端"]
    BI["/bi\nBI Dashboard"]
    Ops["Control Page\n本地控制面"]
  end

  Control --> Collector
  Platforms --> Collector
  Collector --> RawStore
  RawStore --> NATS
  NATS --> Transformer
  Transformer --> Serving
  Transformer --> ClickHouse
  API --> Serving
  API --> ClickHouse
  BI --> API
  Ops --> API
```

## 数据流

```mermaid
sequenceDiagram
  participant CP as control-plane
  participant CW as collector-worker
  participant Raw as raw MySQL
  participant Bus as NATS JetStream
  participant TW as transformer-worker
  participant Store as MySQL / ClickHouse
  participant API as bi-api
  participant FE as React BI

  CP->>CW: dispatch CollectJob
  CW->>Raw: write raw_records and outbox_events
  Raw->>Bus: publish raw event
  Bus->>TW: consume raw event
  TW->>Store: project normalized BI models
  FE->>API: query /api/bi/*
  API->>Store: read BI data
  API-->>FE: return dashboard data
```

## 目录结构

```text
be_ads_project/
  be/                  Go 后端工程
    cmd/               bi-api、collector-worker、transformer-worker、control-plane
    internal/          业务模块和基础设施适配
    scripts/           本地启动、停止、验证脚本
    deploy/            部署和观测配置
    openspec/          后端轻量 SDD
    specs/             后端 Spec Kit SDD

  fe/                  React BI 前端工程
    src/               页面、组件、API client、工具函数
    openspec/          前端轻量 SDD
    specs/             前端 Spec Kit SDD
```

## 快速开始

### Mac 新环境

```bash
git clone git@github.com:nameless422/be-ads-project.git
cd be-ads-project
make mac-start
```

`make mac-start` 会检查 Xcode Command Line Tools、Homebrew、Go、Node.js、Docker Desktop 和本地端口；缺少 Go、Node.js 或 Docker Desktop 时会尝试通过 Homebrew 安装；随后安装前端依赖、启动本地栈并执行验收。

启动成功后访问：

```text
http://127.0.0.1:8080/
http://127.0.0.1:8080/bi
```

### 日常开发

```bash
make up
make verify
make status
make down
```

前端单独开发：

```bash
cd fe
npm ci
npm run dev
```

## 常用命令

```bash
make help              # 查看所有命令
make harness-check     # 仓库级 Harness 检查
make mac-start         # Mac 新环境一键启动
make up                # 启动 MySQL / ClickHouse / NATS 和 4 个后端服务
make verify            # 验收本地主链路
make status            # 查看服务状态和最近日志
make down              # 停止本地栈
make test              # 执行 go test ./...
make frontend-build    # 构建 React 前端
```

## 开发工作流：Harness 驱动 SDD

本仓库不把 Harness 和 SDD 当成两套平行流程。推荐模型是：**Harness 统一接需求、控阶段和做验证；SDD 在 Harness 流程中作为长期设计沉淀被更新**。

| 产物 | 入口 | 在流程里负责什么 |
| --- | --- | --- |
| Harness | [HARNESS.md](HARNESS.md)、[docs/harness](docs/harness/README.md) | 接需求、分阶段、找代码落点、规定验证和 PR gate |
| SDD | [docs/harness/sdd.md](docs/harness/sdd.md)、`be/openspec`、`be/specs`、`fe/openspec`、`fe/specs` | 保存后端、前端和跨模块能力的长期设计决策 |

日常需求默认流程：

1. 先读 [AGENTS.md](AGENTS.md) 和 [HARNESS.md](HARNESS.md)。
2. 在 [docs/harness/playbook.md](docs/harness/playbook.md) 判断任务类型和最少动作。
3. 动代码前查 [docs/harness/dev-map.md](docs/harness/dev-map.md)，确认后端、前端、脚本或文档落点。
4. 涉及 API、字段、数据链路、页面重构或跨前后端联动时，按 [docs/harness/sdd.md](docs/harness/sdd.md) 同步更新对应 SDD。
5. 收尾至少跑 `make harness-check`；改 Go 代码跑 `make test`，改前端跑 `make frontend-build`，改主链路跑 `make verify`。

简单文案、小修和一次性排查可以只走 Harness 记录和验证；复杂需求要把阶段文档放到 `docs/harness/tasks/YYYYMMDD-<slug>/`，并在方案阶段判断是否需要更新对应 SDD。

## API 和页面

主要页面：

```text
/bi
/bi/overview
/bi/breakdown
/bi/creatives
/bi/quality
/bi/control
```

主要 API：

```text
GET  /healthz
GET  /api/bi/snapshots
GET  /api/bi/campaigns
GET  /api/bi/insights/summary
GET  /api/bi/insights/detail
GET  /api/bi/campaign-diagnostics
GET  /api/bi/search-terms
GET  /api/bi/ua-report
GET  /api/bi/ua-fields
GET  /api/bi/game-kpis
POST /api/bi/game-kpis
GET  /api/control/overview
GET  /api/control/dlq
GET  /api/control/local-stack
POST /api/control/local-stack/*
```

## SDD 文档

项目同时使用轻量 OpenSpec 和重流程 Spec Kit：

```text
be/openspec   后端 OpenSpec
be/specs      后端 Spec Kit
fe/openspec   前端 OpenSpec
fe/specs      前端 Spec Kit
```

建议用法：

- 小功能、局部改动：OpenSpec
- 跨模块重构、API/数据链路变化：Spec Kit
- 临时排查和一次性修复：Harness task-board 或 issue 记录

当前已有的主要 SDD：

```text
be/openspec/changes/001-backend-sdd-baseline
be/specs/001-backend-sdd-baseline
be/openspec/changes/002-mac-local-bootstrap
be/specs/002-mac-local-bootstrap
be/specs/003-bi-overview-business-data-foundation
fe/openspec/changes/001-react-bi-refactor
fe/specs/001-react-bi-refactor
fe/openspec/changes/002-bi-overview-user-feedback
```

Harness 不替代 SDD，也不与 SDD 平行抢入口：所有需求先从 Harness 进，只有留下长期接口、字段、链路、页面或运行规则时，才把设计结论同步到 SDD。

SDD 的完整索引和更新规则见 [docs/harness/sdd.md](docs/harness/sdd.md)。

## 更多说明

- [后端说明](be/README.md)
- [前端说明](fe/README.md)
- [Mac 本地启动脚本](be/scripts/ops/mac_local_start.sh)
