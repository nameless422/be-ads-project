# Change: BI Overview Phase 1 UI Alignment

## 背景

这次反馈来自上一版 Overview UI 的用户评审。用户核心意思不是简单调样式，而是希望 Overview 从“后台数据巡检页”变成更贴近 UA 投放业务的看板：先按产品、渠道和日期筛选，再看投放消耗、安装、成本、回收、购买、留存、LTV 等核心指标，并且支持按渠道、地区等维度汇总。

## 分阶段边界

| 阶段 | 文档位置 | 目标 | 边界 |
| --- | --- | --- | --- |
| Phase 1 | `fe/openspec/changes/002-bi-overview-user-feedback` | 只做 Overview 前端职责、布局、文案和已有指标重组。 | 不改后端接口、不改库表、不伪造缺失指标。 |
| Phase 2 | `be/specs/003-bi-overview-business-data-foundation` | 补齐产品维度、D0/D7 回收口径、CPA/CVR 口径和后端数据链路。 | 必须先确认数据来源和业务口径，再改 collector/normalizer/projector/API。 |

## Phase 1 反馈整理

| 模块 | 用户反馈 | 当前系统现状 | Phase 1 处理 |
| --- | --- | --- | --- |
| 筛选命名 | 增加产品名称；渠道字段应叫 `media_source`，不要叫 `platform`。 | 前端 `FilterState`、后端 query、ClickHouse/MySQL 当前都用 `platform` 表示广告来源，如 `facebook`、`google_ads`、`tiktok_ads`。当前没有产品字段。 | UI 先把 `Platform` 展示名改成 `Media Source / media_source`，后端仍保留 `platform` 参数。产品字段进入 Phase 2。 |
| 筛选顺序 | 层级应按产品 -> media_source -> 设备/平台 -> campaign -> adgroup -> ad；开始时间、结束时间单独放。 | 现在顺序是 platform、account、date、entity、country、os、device、network、campaign、ad group、ad、limit、match type、search term。 | Overview 筛选改成两组：日期范围 + 业务层级。`network`、搜索词、匹配类型、limit 从主筛选区移走。 |
| 指标卡 | 不需要这么多卡片，主要看 spend、installs、CPI、ROAS、purchase、CPA、D1 retention、CPM、CTR、CVR、D0 LTV、D7 LTV。 | 现在 Overview 有 10 张卡，包括账户快照、reach/frequency、conversions、revenue、ROI、状态等。部分指标已有，部分可派生，D0 和部分口径需确认。 | Overview 指标卡收敛到这组核心 UA 指标；缺失或口径未确认的指标显示未接入/待确认，不用其它指标冒充。 |
| 趋势图 | 需要按 day 展示折线，指标包括 spend、install、CPI、D0 ROAS、D7 ROAS、D1 retention、CPM、CTR、CVR。 | 当前是 4 个固定趋势：spend、installs、revenue、impressions，没有指标切换，也没有 D0/D7 ROAS。 | 改成“按天趋势”模块，支持已具备数据的指标切换；D0/D7 ROAS 等进入 Phase 2。 |
| 汇总表 | 不要把指标和维度都全列出来；维度可以选择，后面的数都是指标。通常会看分渠道、分地区汇总；账户可以隐藏，或做成可选维度。 | 现在有 Platform Summary、Account Snapshots、UA Overview。Platform Summary 混了库存计数和指标，Account Snapshots 偏运维，UA Overview 是明细表。 | Overview 改成类似 pivot 的汇总表。默认维度为 media_source + country/region，account 默认隐藏，明细表移动到 Breakdown/Quality。 |
| 游戏侧字段 | 广告侧可用，游戏内字段待接入。 | `/api/bi/ua-report` 已经把广告侧 ClickHouse 和游戏侧 MySQL `bi_game_kpis` 合并；没有游戏 KPI 时会显示“广告侧可用，游戏内字段待接入”。 | 保留数据接入状态，但不要压过核心业务指标。install、purchase、retention、LTV 这类指标依赖游戏 KPI 接入质量。 |

## 指标与当前系统映射

| 目标指标 | 当前字段或计算方式 | Phase 1 状态 |
| --- | --- | --- |
| spend | ClickHouse insight `spend`；UA row `spend` | 可展示 |
| installs | Game KPI `installs`；UA row `installs` | 有游戏 KPI 时可展示，否则空态 |
| CPI | `spend / installs`，当前 `UAReportService` 已派生 | installs 有值时可展示 |
| ROAS | UA row `roas` 当前来自广告转化价值 / spend | 可展示，但标注不是 D0/D7 口径 |
| purchase | Game KPI `purchasers` / `purchase_count` | 可展示一个临时口径，最终口径进 Phase 2 |
| CPA | 可用广告 `cost_per_conversion`，也可按 `spend / purchasers` 或 `spend / purchase_count` | 口径未确认，先空态或标注待确认 |
| D1 retention | Game KPI `retention_d1` | 有游戏 KPI 时可展示 |
| CPM | 广告侧 `cpm` | 可展示 |
| CTR | 广告侧 `ctr` | 可展示 |
| CVR | 可按 `installs / clicks` 或 `conversions / clicks` 派生 | 口径未确认，先空态或标注待确认 |
| D0 LTV | 当前没有字段 | Phase 2 |
| D7 LTV | Game KPI `ltv_d7` | 有游戏 KPI 时可展示 |

## Phase 1 纳入范围

- 把 Overview 重新定义为 UA 业务看板，而不是账户/采集状态大杂烩。
- 前端文案上把当前 `platform` 展示为 `media_source`，先不动后端字段名。
- 主 KPI 卡片收敛到用户指定的核心指标。
- 做按天趋势模块，支持已具备数据的指标切换。
- 用可选维度 + 指标值的汇总表替代现在的账号快照/明细表堆叠。
- 明确标注哪些指标依赖游戏侧字段，哪些当前缺失或待确认。

## Phase 1 不纳入范围

- 不直接把后端、库表、领域模型里的 `platform` 全量改名。
- 没有来源定义前，不硬加 `product_name` 假字段。
- 不伪造 D0 LTV、D0 ROAS、D7 ROAS。
- 不改 `/api/bi/*` 响应结构。
- 不删除 Breakdown、Creatives、Quality、Control 这些页面，只调整 Overview 的职责。

## Phase 2 交接问题

- `product_name` 是部署级固定配置、广告账号属性，还是游戏侧数据维度？
- 用户说的“平台”是否指 OS/client platform，还是另一个不同于 `media_source` 的业务维度？
- `purchase` 应展示付费人数、购买次数，还是购买收入？
- `CPA` 应该是 cost per purchaser、cost per purchase event，还是广告平台的 cost per conversion？
- `CVR` 应该是 install CVR（`installs / clicks`）还是广告 conversion CVR（`conversions / clicks`）？
- D0/D7 ROAS 和 D0 LTV 是否需要游戏侧真实字段；如果需要，数据源和写入链路是什么？
