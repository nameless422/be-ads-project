# Raw / Trans / BI 字段梳理

本文基于当前仓库代码整理 `raw -> trans -> bi` 这一条链路里:

- raw 层实际存了哪些字段
- trans 层标准化后存了哪些字段
- BI 层当前展示/返回哪些字段
- 每一层字段是怎么来的
- 当前查询、聚合、派生逻辑是什么

本文只描述当前代码已经实现的部分，不描述未来规划。

## 1. 总体链路

当前链路可以概括为:

1. `collector-worker` 调用各平台 connector 拉平台原始数据。
2. collector 把原始记录写入 `raw mysql.raw_records`，并把事件写入 `raw mysql.outbox_events`。
3. `transformer-worker` 根据 `outbox event` 取回 raw record，按平台 normalizer 转成统一结构。
4. trans 层把结果投影到:
   - `serving mysql` 的维表/快照表
   - `clickhouse` 的明细分析表
5. `bi-api` 再从 `serving mysql` / `clickhouse` 查数，对外提供 `/api/bi/*` 和 `/bi` 页面。

对应主代码位置:

- raw 写入: `internal/modules/collection/application/service.go`
- raw 表结构: `internal/modules/collection/infrastructure/rawstore/mysql/repository.go`
- trans 标准模型: `internal/shared/ads/normalized.go`
- normalizer: `internal/modules/transformation/infrastructure/normalizer/`
- trans 投影:
  - MySQL: `internal/modules/transformation/infrastructure/projector/mysql/projector.go`
  - ClickHouse: `internal/modules/transformation/infrastructure/projector/clickhouse/projector.go`
- BI 查询:
  - MySQL: `internal/modules/reporting/infrastructure/mysql/repository.go`
- ClickHouse: `internal/modules/reporting/infrastructure/clickhouse/repository.go`
  - API/UI: `internal/modules/reporting/infrastructure/httpapi/server.go`

## 1.1 通用字段口径说明

后面很多表和接口会反复出现同名字段。为了避免每一段都重复解释，这里先统一说明一次。

### 维度类字段

| 字段 | 说明 |
| --- | --- |
| `platform` | 广告平台类型，比如 `google_ads`、`facebook`、`tiktok_ads`。 |
| `platform_account_id` | 平台侧账号 ID，是平台 API 语义里的账户标识，也是当前链路里最核心的账号主键。 |
| `account_id` | 系统内部/业务侧账号标识，不一定等于平台账号 ID，更多用于内部归属和查询筛选。 |
| `object_type` | 当前采集对象类型，比如 `campaign`、`ad_group`、`ad`、`insight`、`search_term`。 |
| `resource_id` | 这条 raw 记录在平台侧的对象唯一标识，通常是 campaign/adgroup/ad/search_term 的资源 ID。 |
| `platform_campaign_id` | 平台侧 campaign ID。 |
| `platform_ad_group_id` | 平台侧 ad group / ad set ID。 |
| `platform_ad_id` | 平台侧 ad ID。 |
| `platform_parent_id` | 父级对象 ID。对 ad group 来说通常是 campaign ID，对 ad 来说通常是 ad group ID。 |
| `entity_level` | 指标当前归属粒度，表示这条 insight 是 campaign 级、ad_group 级还是 ad 级。 |
| `entity_id` | 和 `entity_level` 配套使用，表示当前指标实际挂在哪个实体上。 |
| `stat_date` | 指标归属日期，通常是广告平台报表日期，不是入库时间。 |
| `device` | 广告平台返回的设备维度，比如 mobile/desktop/tablet。 |
| `network` | 广告平台返回的网络或流量网络维度，比如 Google Ads 的 `ad_network_type`。 |
| `country` | 国家维度，目前主要用于 UA 游戏 KPI 侧数据。 |
| `os` | 操作系统维度，目前主要用于 UA 游戏 KPI 侧数据。 |
| `placement` | 广告投放版位。 |
| `creative_id` | 创意 ID。 |
| `creative_name` | 创意名称。 |
| `creative_type` | 创意类型，比如视频、图片等。 |
| `optimization_goal` | 优化目标，例如安装、激活、注册等。 |
| `bid_type` | 出价类型。 |
| `targeting` | 定向配置描述，当前以文本方式保存。 |
| `search_term` | 用户实际触发广告的搜索词。 |
| `search_term_match_type` | 搜索词匹配方式，比如 exact/phrase。 |

### 指标类字段

| 字段 | 说明 |
| --- | --- |
| `impressions` | 展现次数。 |
| `clicks` | 点击次数。 |
| `spend` | 消耗金额。当前链路里大多以字符串十进制金额表示，落 ClickHouse/MySQL 后再转数值。 |
| `ctr` | 点击率，通常是 `clicks / impressions * 100`。 |
| `cpc` | 单次点击成本，通常是 `spend / clicks`。 |
| `cpm` | 千次展现成本，通常是 `spend / impressions * 1000`。 |
| `conversions` | 转化次数，口径取决于平台返回。 |
| `all_conversions` | 全部转化次数，通常比 `conversions` 更宽口径，目前主要来自 Google Ads。 |
| `conversions_value` | 转化价值金额。 |
| `cost_per_conversion` | 单次转化成本，通常是 `spend / conversions`。 |
| `cost_per_all_conversions` | 全部转化口径下的单次转化成本，通常是 `spend / all_conversions`。 |
| `reach` | 去重触达人数。 |
| `frequency` | 人均触达频次，通常是 `impressions / reach`。 |
| `roas` | 广告回报率，通常是 `conversions_value / spend`。 |
| `installs` | 安装数。 |
| `activations` | 激活数。 |
| `registrations` | 注册数。 |
| `tutorial_completions` | 新手教程完成数。 |
| `role_creations` | 建角数。 |
| `level_x_users` | 到达某关键等级的人数。 |
| `purchasers` | 付费人数。 |
| `purchase_count` | 付费次数。 |
| `first_purchase_amount` | 首购金额。 |
| `revenue_d1 / revenue_d7 / revenue_d30` | 1/7/30 日收入。 |
| `ad_revenue` | 广告变现收入。 |
| `total_revenue` | 总收入。 |
| `retention_d1 / retention_d3 / retention_d7 / retention_d30` | 次日/3日/7日/30日留存率。 |
| `ltv_d7 / ltv_d30` | 7日/30日 LTV。 |
| `avg_online_duration_seconds` | 平均在线时长，单位秒。 |
| `task_completion_rate` | 任务完成率。 |
| `high_value_payer_ratio` | 高价值付费用户占比。 |

### 链路/维护类字段

| 字段 | 说明 |
| --- | --- |
| `job_id` | 一次采集任务的唯一标识。 |
| `trace_id` | 链路追踪 ID，用于把调度、采集、转换串起来。 |
| `profile_id` | 同步配置的 profile ID，决定平台、对象类型、账号等。 |
| `source_mode` | 本次采集的运行模式，比如 mock、seeded_test、jetstream_async。 |
| `source_cursor` | 平台分页游标。 |
| `source_watermark` | 增量同步使用的时间水位或业务水位。 |
| `collected_at` | 采集动作完成时记录的时间。 |
| `created_at / updated_at` | 记录创建/更新时间。 |
| `ingested_at` | trans 投影写入下游库的时间。 |
| `source_updated_at` | 平台原始对象在广告平台上的最近更新时间。 |
| `raw_payload` / `payload_json` | 原始 JSON 内容，用于排障、回溯和后续扩字段。 |

## 2. Raw 层

### 2.1 raw mysql 的表

当前 raw 层核心有两张表:

#### `raw_records`

表结构来自 `internal/modules/collection/infrastructure/rawstore/mysql/repository.go`:

| 字段 | 含义 | 来源 |
| --- | --- | --- |
| `id` | 自增主键 | MySQL |
| `job_id` | 本次采集任务 ID | collect job |
| `trace_id` | 链路追踪 ID | collect job |
| `profile_id` | 同步 profile | collect job |
| `platform` | 平台 | collect job / raw record |
| `platform_account_id` | 平台账号 ID | raw record |
| `account_id` | 业务账号 ID | collect job |
| `object_type` | 对象类型 | raw record |
| `resource_id` | 平台对象资源 ID | raw record |
| `payload_json` | 原始 JSON payload | connector 返回的 raw payload |
| `source_mode` | 采集模式 | collector 产出 batch 的 `SourceMode` |
| `source_cursor` | 分页 cursor | collector 产出 batch 的 `NextCursor` |
| `source_watermark` | 水位信息 | collect job |
| `collected_at` | 平台数据采集时间 | collector batch |
| `created_at` | 入库时间 | 写库时 `now` |

#### `outbox_events`

同文件里还会把 raw batch 对应事件写入 `outbox_events`:

| 字段 | 含义 |
| --- | --- |
| `event_id` | 事件 ID |
| `aggregate_type` | 这里固定是 `raw_batch` |
| `aggregate_id` | 对应 `job_id` |
| `event_type` | 这里固定是 `raw.ingested` |
| `topic` | JetStream subject |
| `payload_json` | `messagingdomain.RawEvent` |
| `status` | `pending/failed/published` |
| `attempt_count` | 发布重试次数 |
| `last_error` | 上一次失败原因 |
| `available_at` | 下次允许投递时间 |
| `published_at` | 成功发布时间 |
| `created_at/updated_at` | 维护字段 |

### 2.2 raw 是怎么写进去的

入口在 `internal/modules/collection/application/service.go` 的 `HandleJob`:

1. 先通过 `resolver.ResolveTarget` 拿到 profile / account / credential。
2. 再通过 `collector.Collect` 调平台 connector。
3. connector 返回 `CollectedBatch`:
   - `Records`
   - `SourceMode`
   - `NextCursor`
   - `CollectedAtUTC`
4. collector 把 `Records` 原样写入 `raw_records`。
5. 同事务写一条 `outbox_event`，payload 里带 `RawRecordIDs`。

注意:

- raw 层没有做字段标准化，`payload_json` 基本就是 connector 返回的原始对象 JSON。
- transformer 后续取数时，会用 `GetByIDs()` 从 `raw_records` 读回:
  - `platform`
  - `platform_account_id`
  - `object_type`
  - `resource_id`
  - `payload_json`

### 2.3 raw payload 目前按平台/对象有哪些字段

当前 raw payload 的“实际可消费字段”不是通过一个统一 schema 管理，而是由各平台 normalizer 直接解码。也就是说，下面这些字段就是当前 trans 层真正依赖的 raw 字段。

这里还要注意一个现实差异:

- 不同平台同名字段未必同口径。
- 不同平台很多字段虽然最后都映射到统一模型里的同一个字段，但它们在平台原语义上只是“近似对齐”，不是严格等价。
- 当前文档描述的是“代码里怎么映射”，不是“平台官方语义已经完全统一”。

#### Google Ads

来源:

- 实时采集: `internal/modules/collection/infrastructure/connectors/googleads/real_client.go`
- 标准化消费: `internal/modules/transformation/infrastructure/normalizer/googleads_normalizer.go`

##### `object_type=campaign`

raw payload 当前会被消费这些字段:

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `campaign.id` | campaign ID | `StandardCampaign.PlatformCampaignID` |
| `campaign.name` | campaign 名称 | `StandardCampaign.CampaignName` |
| `campaign.status` | campaign 状态 | `StandardCampaign.Status` |
| `advertising_channel_type` | 广告渠道/营销目标类型 | `StandardCampaign.Objective` |
| `start_date` | campaign 开始日期 | `StandardCampaign.StartTime` |
| `end_date` | campaign 结束日期 | `StandardCampaign.EndTime` |
| `bidding_strategy_type` | 出价策略类型 | `StandardCampaign.BuyingType` 和 `StandardCampaign.BiddingStrategy` |
| `campaign_budget` | campaign 预算 | `StandardCampaign.DailyBudget` |
| `updated_at` | 平台对象更新时间 | `StandardCampaign.UpdatedAt` |
| `diagnostics.date` | 诊断指标日期 | `StandardCampaignDiagnostic.StatDate` |
| `diagnostics.search_impression_share` | 搜索展示份额 | `StandardCampaignDiagnostic.SearchImpressionShare` |
| `diagnostics.search_top_impression_share` | 搜索顶部展示份额 | `StandardCampaignDiagnostic.SearchTopImpressionShare` |
| `diagnostics.search_absolute_top_impression_share` | 搜索绝对顶部展示份额 | `StandardCampaignDiagnostic.SearchAbsoluteTopImpressionShare` |

##### `object_type=ad_group`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `ad_group.id` | ad group ID | `StandardAdGroup.PlatformAdGroupID` |
| `ad_group.name` | ad group 名称 | `StandardAdGroup.AdGroupName` |
| `ad_group.status` | ad group 状态 | `StandardAdGroup.Status` |
| `campaign_id` | 所属 campaign ID | `StandardAdGroup.PlatformParentID` |
| `cpc_bid` | CPC 出价值 | `StandardAdGroup.BidStrategy` |
| `updated_at` | 平台对象更新时间 | `StandardAdGroup.UpdatedAt` |

##### `object_type=ad`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `ad_group_ad.status` | 广告状态 | `StandardAd.Status` |
| `ad_group_ad.ad.id` | ad ID | `StandardAd.PlatformAdID` |
| `ad_group_ad.ad.name` | ad 名称 | `StandardAd.AdName` |
| `ad_group_id` | 所属 ad group ID | `StandardAd.PlatformParentID` |
| `updated_at` | 平台对象更新时间 | `StandardAd.UpdatedAt` |

##### `object_type=insight`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `campaign_id` | campaign ID | `StandardInsight.PlatformCampaignID` |
| `ad_group_id` | ad group ID | `StandardInsight.PlatformAdGroupID`，也参与 `EntityLevel/EntityID` 判断 |
| `ad_id` | ad ID | `StandardInsight.PlatformAdID`，也参与 `EntityLevel/EntityID` 判断 |
| `segments.date` | 报表日期 | `StandardInsight.StatDate` |
| `segments.device` | 设备维度 | `StandardInsight.Device` |
| `segments.ad_network_type` | 广告网络类型 | `StandardInsight.Network` |
| `metrics.impressions` | 展现次数 | `StandardInsight.Impressions` |
| `metrics.clicks` | 点击次数 | `StandardInsight.Clicks` |
| `metrics.cost_micros` | 微单位消耗金额 | 先转十进制，再映射到 `StandardInsight.Spend` |
| `metrics.ctr` | 点击率 | `StandardInsight.CTR` |
| `metrics.average_cpc` | 平均 CPC | `StandardInsight.CPC` |
| `metrics.average_cpm` | 平均 CPM | `StandardInsight.CPM` |
| `metrics.conversions` | 转化次数 | `StandardInsight.Conversions` |
| `metrics.all_conversions` | 全部转化次数 | `StandardInsight.AllConversions` |
| `metrics.conversions_value` | 转化价值 | `StandardInsight.ConversionsValue` |
| `metrics.cost_per_conversion` | 单次转化成本 | `StandardInsight.CostPerConversion` |
| `metrics.cost_per_all_conversions` | 全部转化口径单次成本 | `StandardInsight.CostPerAllConversions` |

##### `object_type=search_term`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `campaign_id` | campaign ID | `StandardSearchTerm.PlatformCampaignID` |
| `ad_group_id` | ad group ID | `StandardSearchTerm.PlatformAdGroupID` |
| `search_term` | 实际搜索词 | `StandardSearchTerm.SearchTerm` |
| `segments.date` | 报表日期 | `StandardSearchTerm.StatDate` |
| `segments.search_term_match_type` | 匹配类型 | `StandardSearchTerm.SearchTermMatchType` |
| `metrics.impressions` | 展现次数 | `StandardSearchTerm.Impressions` |
| `metrics.clicks` | 点击次数 | `StandardSearchTerm.Clicks` |
| `metrics.cost_micros` | 微单位消耗金额 | 先转十进制，再映射到 `StandardSearchTerm.Spend` |
| `metrics.conversions` | 转化次数 | `StandardSearchTerm.Conversions` |
| `metrics.conversions_value` | 转化价值 | `StandardSearchTerm.ConversionsValue` |

#### Facebook

来源:

- connector: `internal/modules/collection/infrastructure/connectors/facebook/connector.go`
- normalizer: `internal/modules/transformation/infrastructure/normalizer/facebook_normalizer.go`

##### `object_type=campaign`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `id` | campaign ID | `StandardCampaign.PlatformCampaignID` |
| `name` | campaign 名称 | `StandardCampaign.CampaignName` |
| `effective_status` | 生效状态 | `StandardCampaign.Status` |
| `objective` | 投放目标 | `StandardCampaign.Objective` |
| `buying_type` | 购买类型 | `StandardCampaign.BuyingType` |
| `daily_budget` | 日预算 | `StandardCampaign.DailyBudget` |
| `currency` | 币种 | `StandardCampaign.Currency` |
| `updated_time` | 平台对象更新时间 | `StandardCampaign.UpdatedAt` |

##### `object_type=ad_group`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `id` | ad group / ad set ID | `StandardAdGroup.PlatformAdGroupID` |
| `campaign_id` | 所属 campaign ID | `StandardAdGroup.PlatformParentID` |
| `name` | ad group 名称 | `StandardAdGroup.AdGroupName` |
| `status` | ad group 状态 | `StandardAdGroup.Status` |
| `bid_strategy` | 出价策略 | `StandardAdGroup.BidStrategy` |
| `daily_budget` | 日预算 | `StandardAdGroup.DailyBudget` |
| `updated_time` | 平台对象更新时间 | `StandardAdGroup.UpdatedAt` |

##### `object_type=ad`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `id` | ad ID | `StandardAd.PlatformAdID` |
| `adset_id` | 所属 ad set ID | `StandardAd.PlatformParentID` |
| `name` | ad 名称 | `StandardAd.AdName` |
| `status` | 广告状态 | `StandardAd.Status` |
| `updated_time` | 平台对象更新时间 | `StandardAd.UpdatedAt` |
| `creative.id` | 创意 ID | `StandardAd.CreativeID` |
| `creative.name` | 创意名称 | `StandardAd.CreativeName` |

##### `object_type=insight`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `campaign_id` | campaign ID | `StandardInsight.PlatformCampaignID`，同时作为 campaign 级 `EntityID` |
| `date_start` | 报表日期 | `StandardInsight.StatDate` |
| `impressions` | 展现次数 | `StandardInsight.Impressions` |
| `clicks` | 点击次数 | `StandardInsight.Clicks` |
| `spend` | 消耗金额 | `StandardInsight.Spend` |
| `ctr` | 点击率 | `StandardInsight.CTR` |
| `cpc` | 单次点击成本 | `StandardInsight.CPC` |
| `cpm` | 千次展现成本 | `StandardInsight.CPM` |
| `conversions` | 转化次数 | `StandardInsight.Conversions` |
| `reach` | 触达人数 | `StandardInsight.Reach` |

#### TikTok Ads

来源:

- connector: `internal/modules/collection/infrastructure/connectors/tiktok/connector.go`
- normalizer: `internal/modules/transformation/infrastructure/normalizer/tiktok_normalizer.go`

##### `object_type=campaign`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `campaign_id` | campaign ID | `StandardCampaign.PlatformCampaignID` |
| `campaign_name` | campaign 名称 | `StandardCampaign.CampaignName` |
| `operation_status` | 运营状态 | `StandardCampaign.Status` |
| `objective_type` | 投放目标 | `StandardCampaign.Objective` |
| `budget_mode` | 预算模式 | `StandardCampaign.BuyingType` |
| `budget` | 预算金额 | `StandardCampaign.DailyBudget` |
| `modify_time` | 平台对象更新时间 | `StandardCampaign.UpdatedAt` |

##### `object_type=ad_group`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `adgroup_id` | ad group ID | `StandardAdGroup.PlatformAdGroupID` |
| `campaign_id` | 所属 campaign ID | `StandardAdGroup.PlatformParentID` |
| `adgroup_name` | ad group 名称 | `StandardAdGroup.AdGroupName` |
| `operation_status` | ad group 状态 | `StandardAdGroup.Status` |
| `bid_type` | 出价类型 | `StandardAdGroup.BidStrategy` |
| `budget` | 预算金额 | `StandardAdGroup.DailyBudget` |
| `modify_time` | 平台对象更新时间 | `StandardAdGroup.UpdatedAt` |

##### `object_type=ad`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `ad_id` | ad ID | `StandardAd.PlatformAdID` |
| `adgroup_id` | 所属 ad group ID | `StandardAd.PlatformParentID` |
| `ad_name` | ad 名称 | `StandardAd.AdName` |
| `operation_status` | 广告状态 | `StandardAd.Status` |
| `creative_id` | 创意 ID | `StandardAd.CreativeID` |
| `modify_time` | 平台对象更新时间 | `StandardAd.UpdatedAt` |

##### `object_type=insight`

| 字段 | 中文含义 | 后续映射 |
| --- | --- | --- |
| `campaign_id` | campaign ID | `StandardInsight.PlatformCampaignID`，同时作为 campaign 级 `EntityID` |
| `stat_time_day` | 报表日期 | `StandardInsight.StatDate` |
| `metrics.show_cnt` | 展现次数 | `StandardInsight.Impressions` |
| `metrics.click_cnt` | 点击次数 | `StandardInsight.Clicks` |
| `metrics.stat_cost` | 消耗金额 | `StandardInsight.Spend` |
| `metrics.ctr` | 点击率 | `StandardInsight.CTR` |
| `metrics.cpc` | 单次点击成本 | `StandardInsight.CPC` |
| `metrics.cpm` | 千次展现成本 | `StandardInsight.CPM` |
| `metrics.convert_cnt` | 转化次数 | `StandardInsight.Conversions` |
| `metrics.reach` | 触达人数 | `StandardInsight.Reach` |

### 2.4 raw 层当前的边界

有两个地方需要特别说明:

1. `account` 类型虽然在对象类型枚举和 collector 分发里存在，但当前 normalizer 基本没有把 `raw account payload` 真正投影到后续 BI。
2. raw payload 并不是“平台完整响应字段全集”；它是“当前 connector 产出的 JSON”，而后续 trans 层只会读取它认得的字段。

## 3. Trans 层

## 3.1 trans 的统一标准模型

标准模型定义在 `internal/shared/ads/normalized.go`。

当前 trans 层的统一结构有:

### `StandardCampaign`

| 字段 | 说明 |
| --- | --- |
| `Platform` | campaign 所属平台。 |
| `PlatformAccountID` | campaign 所属平台账号。 |
| `PlatformCampaignID` | 平台侧 campaign 主键。 |
| `CampaignName` | campaign 名称。 |
| `Status` | campaign 当前状态。 |
| `Objective` | 投放目标/营销目标。 |
| `BuyingType` | 购买类型或预算模式。不同平台字段名不同，这里统一收口。 |
| `BiddingStrategy` | 出价策略。 |
| `DailyBudget` | 日预算。 |
| `LifetimeBudget` | 生命周期总预算。当前主要是有些平台会给，有些为空。 |
| `Currency` | 预算和金额字段的币种。 |
| `StartTime` | campaign 开始时间。 |
| `EndTime` | campaign 结束时间。 |
| `UpdatedAt` | 平台原对象最近更新时间。 |
| `RawPayload` | 对应 raw 原文，方便排障和补字段。 |

### `StandardCampaignDiagnostic`

| 字段 | 说明 |
| --- | --- |
| `Platform` | 诊断数据所属平台。 |
| `PlatformAccountID` | 所属平台账号。 |
| `PlatformCampaignID` | 所属 campaign。 |
| `StatDate` | 诊断指标日期。 |
| `SearchImpressionShare` | 搜索展示份额。 |
| `SearchTopImpressionShare` | 搜索顶部展示份额。 |
| `SearchAbsoluteTopImpressionShare` | 搜索绝对顶部展示份额。 |
| `RawPayload` | 原始 payload。 |

### `StandardAdGroup`

| 字段 | 说明 |
| --- | --- |
| `Platform` | ad group 所属平台。 |
| `PlatformAccountID` | ad group 所属平台账号。 |
| `PlatformAdGroupID` | 平台侧 ad group 主键。 |
| `PlatformParentID` | 父级 campaign ID。 |
| `AdGroupName` | ad group 名称。 |
| `Status` | ad group 状态。 |
| `BidStrategy` | ad group 级别的出价策略或出价值。 |
| `DailyBudget` | ad group 日预算。 |
| `StartTime` | ad group 开始时间。 |
| `EndTime` | ad group 结束时间。 |
| `UpdatedAt` | 平台原对象更新时间。 |
| `RawPayload` | 原始 payload。 |

### `StandardAd`

| 字段 | 说明 |
| --- | --- |
| `Platform` | ad 所属平台。 |
| `PlatformAccountID` | ad 所属平台账号。 |
| `PlatformAdID` | 平台侧 ad 主键。 |
| `PlatformParentID` | 父级 ad group ID。 |
| `AdName` | 广告名称。 |
| `Status` | 广告状态。 |
| `CreativeID` | 关联创意 ID。 |
| `CreativeName` | 关联创意名称。 |
| `UpdatedAt` | 平台原对象更新时间。 |
| `RawPayload` | 原始 payload。 |

### `StandardInsight`

| 字段 | 说明 |
| --- | --- |
| `Platform` | 指标所属平台。 |
| `PlatformAccountID` | 指标所属平台账号。 |
| `PlatformCampaignID` | 指标关联 campaign。 |
| `EntityLevel` | 当前指标归属粒度，campaign/ad_group/ad。 |
| `EntityID` | 与 `EntityLevel` 对应的实体 ID。 |
| `PlatformAdGroupID` | 指标关联 ad group。没有时可为空。 |
| `PlatformAdID` | 指标关联 ad。没有时可为空。 |
| `StatDate` | 报表日期。 |
| `Device` | 设备维度。 |
| `Network` | 网络维度。 |
| `Impressions` | 展现次数。 |
| `Clicks` | 点击次数。 |
| `Spend` | 消耗金额。 |
| `CTR` | 点击率。 |
| `CPC` | 单次点击成本。 |
| `CPM` | 千次展现成本。 |
| `Conversions` | 转化次数。 |
| `AllConversions` | 全部转化次数。 |
| `ConversionsValue` | 转化价值。 |
| `CostPerConversion` | 单次转化成本。 |
| `CostPerAllConversions` | 全部转化口径下的单次转化成本。 |
| `Reach` | 触达人数。 |
| `RawPayload` | 原始 payload。 |

### `StandardSearchTerm`

| 字段 | 说明 |
| --- | --- |
| `Platform` | 搜索词数据所属平台。 |
| `PlatformAccountID` | 所属平台账号。 |
| `PlatformCampaignID` | 所属 campaign。 |
| `PlatformAdGroupID` | 所属 ad group。 |
| `SearchTerm` | 触发广告的实际搜索词。 |
| `SearchTermMatchType` | 搜索词匹配类型。 |
| `StatDate` | 报表日期。 |
| `Impressions` | 该搜索词的展现次数。 |
| `Clicks` | 该搜索词的点击次数。 |
| `Spend` | 该搜索词消耗金额。 |
| `Conversions` | 该搜索词带来的转化次数。 |
| `ConversionsValue` | 该搜索词带来的转化价值。 |
| `RawPayload` | 原始 payload。 |

## 3.2 trans 是怎么来的

入口在 `internal/modules/transformation/application/worker.go` + `service.go`:

1. transformer 消费 `RawEvent`。
2. 用 `RawRecordIDs` 回 raw mysql 取记录。
3. 按 `platform` 走对应 normalizer:
   - `GoogleAdsNormalizer`
   - `FacebookNormalizer`
   - `TikTokNormalizer`
4. normalizer 把 raw JSON 解码为统一标准结构。
5. 再分别投影到 MySQL / ClickHouse / BI snapshot。

几个关键转换逻辑:

- 字符串时间通过 `parseRFC3339Ptr` / `parseDatePtr` / `parseDate` 转成 `time.Time`
- 字符串整型通过 `parseInt64` 转成 `int64`
- Google Ads 的 `cost_micros` 通过 `microsToDecimalString` 转成普通十进制金额字符串
- `RawPayload` 会继续保留下去，既会进入 serving mysql，也会进入 clickhouse

## 3.3 trans 投影到 serving mysql 的字段

### `oltp_accounts`

表结构在 `internal/modules/transformation/infrastructure/projector/mysql/projector.go`:

| 字段 | 来源 |
| --- | --- |
| `platform_account_id` | `account.ID` |
| `platform` | `account.Platform` |
| `account_id` | `account.AccountID` |
| `account_name` | `account.AccountName` |
| `status` | `account.Status` |
| `timezone` | `account.Timezone` |
| `currency` | `account.Currency` |
| `raw_payload` | 当前没有从 normalizer 写入，账户维度主要来自 target bundle |
| `ingested_at` | `batch.NormalizedAt` |
| `updated_at` | `batch.NormalizedAt` |

注意:

- 账户维度这里主要取的是 `batch.Collected.Target.Bundle.Account`，不是消费 `StandardAccount`。
- 当前 trans normalizer 基本没有实际产出 account 维度列表给 projector 使用。

### `oltp_campaigns`

| 字段 | 来源 |
| --- | --- |
| `platform_account_id` | `StandardCampaign.PlatformAccountID` |
| `platform_campaign_id` | `StandardCampaign.PlatformCampaignID` |
| `platform` | `StandardCampaign.Platform` |
| `account_id` | `target bundle account.AccountID` |
| `campaign_name` | `CampaignName` |
| `status` | `Status` |
| `objective` | `Objective` |
| `buying_type` | `BuyingType` |
| `bidding_strategy` | `BiddingStrategy` |
| `daily_budget` | `DailyBudget` |
| `lifetime_budget` | `LifetimeBudget` |
| `currency` | `Currency` |
| `start_time` | `StartTime` |
| `end_time` | `EndTime` |
| `source_updated_at` | `UpdatedAt` |
| `raw_payload` | `RawPayload` |
| `ingested_at` | `batch.NormalizedAt` |

### `oltp_ad_groups`

| 字段 | 来源 |
| --- | --- |
| `platform_account_id` | `PlatformAccountID` |
| `platform_ad_group_id` | `PlatformAdGroupID` |
| `platform` | `Platform` |
| `account_id` | `target bundle account.AccountID` |
| `platform_parent_id` | `PlatformParentID` |
| `ad_group_name` | `AdGroupName` |
| `status` | `Status` |
| `bid_strategy` | `BidStrategy` |
| `daily_budget` | `DailyBudget` |
| `start_time` | `StartTime` |
| `end_time` | `EndTime` |
| `source_updated_at` | `UpdatedAt` |
| `raw_payload` | `RawPayload` |
| `ingested_at` | `batch.NormalizedAt` |

### `oltp_ads`

| 字段 | 来源 |
| --- | --- |
| `platform_account_id` | `PlatformAccountID` |
| `platform_ad_id` | `PlatformAdID` |
| `platform` | `Platform` |
| `account_id` | `target bundle account.AccountID` |
| `platform_parent_id` | `PlatformParentID` |
| `ad_name` | `AdName` |
| `status` | `Status` |
| `creative_id` | `CreativeID` |
| `creative_name` | `CreativeName` |
| `source_updated_at` | `UpdatedAt` |
| `raw_payload` | `RawPayload` |
| `ingested_at` | `batch.NormalizedAt` |

## 3.4 trans 投影到 ClickHouse 的字段

### `olap_insights`

表结构在 `internal/modules/transformation/infrastructure/projector/clickhouse/projector.go`:

| 字段 | 来源 |
| --- | --- |
| `platform` | `StandardInsight.Platform` |
| `platform_account_id` | `PlatformAccountID` |
| `platform_campaign_id` | `PlatformCampaignID` |
| `entity_level` | `EntityLevel` |
| `entity_id` | `EntityID` |
| `platform_ad_group_id` | `PlatformAdGroupID` |
| `platform_ad_id` | `PlatformAdID` |
| `stat_date` | `StatDate` |
| `device` | `Device` |
| `network` | `Network` |
| `impressions` | `Impressions` |
| `clicks` | `Clicks` |
| `spend` | `Spend`，写入前经 `decimalString` 清洗 |
| `ctr` | `CTR` |
| `cpc` | `CPC` |
| `cpm` | `CPM` |
| `conversions` | `Conversions` |
| `all_conversions` | `AllConversions` |
| `conversions_value` | `ConversionsValue` |
| `cost_per_conversion` | `CostPerConversion` |
| `cost_per_all_conversions` | `CostPerAllConversions` |
| `reach` | `Reach` |
| `raw_payload` | `RawPayload` 的字符串形式 |
| `ingested_at` | `batch.NormalizedAt` |

### `olap_campaign_diagnostics`

| 字段 | 来源 |
| --- | --- |
| `platform` | `StandardCampaignDiagnostic.Platform` |
| `platform_account_id` | `PlatformAccountID` |
| `platform_campaign_id` | `PlatformCampaignID` |
| `stat_date` | `StatDate` |
| `search_impression_share` | `SearchImpressionShare` |
| `search_top_impression_share` | `SearchTopImpressionShare` |
| `search_absolute_top_impression_share` | `SearchAbsoluteTopImpressionShare` |
| `raw_payload` | `RawPayload` |
| `ingested_at` | `batch.NormalizedAt` |

### `olap_search_terms`

| 字段 | 来源 |
| --- | --- |
| `platform` | `StandardSearchTerm.Platform` |
| `platform_account_id` | `PlatformAccountID` |
| `platform_campaign_id` | `PlatformCampaignID` |
| `platform_ad_group_id` | `PlatformAdGroupID` |
| `search_term` | `SearchTerm` |
| `search_term_match_type` | `SearchTermMatchType` |
| `stat_date` | `StatDate` |
| `impressions` | `Impressions` |
| `clicks` | `Clicks` |
| `spend` | `Spend` |
| `conversions` | `Conversions` |
| `conversions_value` | `ConversionsValue` |
| `raw_payload` | `RawPayload` |
| `ingested_at` | `batch.NormalizedAt` |

## 3.5 trans 还会写 BI snapshot

`internal/modules/reporting/application/service.go` 会把每个 normalized batch 合并到 `bi_account_snapshots`。

当前 snapshot 字段:

| 字段 | 来源/逻辑 |
| --- | --- |
| `platform_account_id` | target bundle account |
| `platform` | target bundle account |
| `account_id` | target bundle account |
| `account_name` | target bundle account |
| `last_source_mode` | 当前 batch 的 `SourceMode` |
| `last_object_type` | 当前 profile 的对象类型 |
| `last_collected_at` | 当前 batch 的 `CollectedAtUTC` |
| `campaign_count` | 如果本批是 `campaign`，直接覆盖成 `len(payload.Campaigns)` |
| `ad_group_count` | 如果本批是 `ad_group`，直接覆盖成 `len(payload.AdGroups)` |
| `ad_count` | 如果本批是 `ad`，直接覆盖成 `len(payload.Ads)` |
| `insight_count` | 如果本批是 `insight`，直接覆盖成 `len(payload.Insights)` |

这里要注意:

- snapshot 不是实时聚合维表全量行数，而是“某账号最近一次处理到各对象类型时记录下来的数量”。
- 只对 `campaign/ad_group/ad/insight` 做计数更新。
- 目前没有 `search_term_count` 字段。

## 4. BI 层

## 4.1 BI 当前有哪些数据源

### 来自 serving mysql

- `bi_account_snapshots`
- `oltp_campaigns`
- `bi_game_kpis`

### 来自 clickhouse

- `olap_insights`
- `olap_campaign_diagnostics`
- `olap_search_terms`

### BI API / 页面入口

定义在 `internal/modules/reporting/infrastructure/httpapi/server.go`:

- `GET /api/bi/snapshots`
- `GET /api/bi/campaigns`
- `GET /api/bi/insights/summary`
- `GET /api/bi/insights/detail`
- `GET /api/bi/campaign-diagnostics`
- `GET /api/bi/search-terms`
- `GET /api/bi/ua-report`
- `GET /api/bi/ua-fields`
- `GET /api/bi/game-kpis`
- `GET /bi`

## 4.2 各 BI 接口返回字段和取数逻辑

### 4.2.1 `/api/bi/snapshots`

来源: `bi_account_snapshots`

字段解释补充:

- `last_source_mode`: 最近一次更新这个 snapshot 的采集来源模式。
- `last_object_type`: 最近一次刷新的是哪一类对象。
- `campaign_count/ad_group_count/ad_count/insight_count`: 最近一次对应对象批次写入后记录下来的数量，不是实时全库 count。

返回字段:

- `platform`
- `platform_account_id`
- `account_id`
- `account_name`
- `last_source_mode`
- `last_object_type`
- `last_collected_at`
- `campaign_count`
- `ad_group_count`
- `ad_count`
- `insight_count`

逻辑:

- 直接 `SELECT ... FROM bi_account_snapshots ORDER BY platform, account_id`
- 不做额外聚合

### 4.2.2 `/api/bi/campaigns`

来源: `oltp_campaigns`

字段解释补充:

- `buying_type`: 平台购买模式或预算模式。
- `bidding_strategy`: 平台出价策略。
- `source_updated_at`: 平台原对象更新时间。
- `ingested_at`: trans 写入 serving mysql 的时间。

返回字段:

- `platform`
- `platform_account_id`
- `account_id`
- `platform_campaign_id`
- `campaign_name`
- `status`
- `objective`
- `buying_type`
- `bidding_strategy`
- `daily_budget`
- `lifetime_budget`
- `currency`
- `start_time`
- `end_time`
- `source_updated_at`
- `ingested_at`

过滤条件:

- `platform`
- `account_id`

逻辑:

- 直接从 `oltp_campaigns` 查
- 按 `platform, account_id, platform_campaign_id` 排序

### 4.2.3 `/api/bi/insights/summary`

来源: `clickhouse.olap_insights`

字段解释补充:

- 这是“按账号 + 日期”汇总后的日级汇总数据，不再区分 campaign/adgroup/ad。
- `cost_per_conversion` 和 `cost_per_all_conversions` 是查询时动态算出来的，不是直接取单行原值。

返回字段:

- `platform`
- `platform_account_id`
- `stat_date`
- `impressions`
- `clicks`
- `spend`
- `conversions`
- `all_conversions`
- `conversions_value`
- `cost_per_conversion`
- `cost_per_all_conversions`
- `reach`

过滤条件:

- `platform`
- `platform_account_id`
- `date_from`
- `date_to`

逻辑:

- 在 `olap_insights FINAL` 上按:
  - `platform`
  - `platform_account_id`
  - `stat_date`
  聚合
- 聚合方式:
  - `sum(impressions)`
  - `sum(clicks)`
  - `sum(spend)`
  - `sum(conversions)`
  - `sum(all_conversions)`
  - `sum(conversions_value)`
  - `sum(reach)`
- 派生:
  - `cost_per_conversion = spend / conversions`
  - `cost_per_all_conversions = spend / all_conversions`
  - 分母为 0 时返回 0

### 4.2.4 `/api/bi/insights/detail`

来源: `clickhouse.olap_insights`

字段解释补充:

- 这是指标明细视图，保留了 `entity_level/entity_id/device/network` 粒度。
- `platform_campaign_id/platform_ad_group_id/platform_ad_id` 用来保留上下级归属，便于下钻。

返回字段:

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `entity_level`
- `entity_id`
- `platform_ad_group_id`
- `platform_ad_id`
- `stat_date`
- `device`
- `network`
- `impressions`
- `clicks`
- `spend`
- `ctr`
- `cpc`
- `cpm`
- `conversions`
- `all_conversions`
- `conversions_value`
- `cost_per_conversion`
- `cost_per_all_conversions`
- `reach`

过滤条件:

- `platform`
- `platform_account_id`
- `date_from`
- `date_to`
- `entity_level`
- `device`
- `network`
- `limit`

逻辑:

- 直接查 `olap_insights FINAL`
- 不做二次聚合，取已经投影进去的明细行
- 按 `stat_date DESC, platform, platform_account_id, entity_level, entity_id` 排序

### 4.2.5 `/api/bi/campaign-diagnostics`

来源: `clickhouse.olap_campaign_diagnostics`

字段解释补充:

- 这几个 share 指标都是 Google Ads 搜索广告诊断型指标，目前主要用于看展示份额和顶部占比。

返回字段:

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `stat_date`
- `search_impression_share`
- `search_top_impression_share`
- `search_absolute_top_impression_share`

过滤条件:

- `platform`
- `platform_account_id`
- `date_from`
- `date_to`
- `limit`

逻辑:

- 直接查 `olap_campaign_diagnostics FINAL`
- 不做额外聚合

### 4.2.6 `/api/bi/search-terms`

来源: `clickhouse.olap_search_terms`

字段解释补充:

- 一行代表一个搜索词在某日、某账号、某 campaign/adgroup 下的表现。
- 适合做关键词挖掘、无效词筛查、品牌词观察。

返回字段:

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `platform_ad_group_id`
- `search_term`
- `search_term_match_type`
- `stat_date`
- `impressions`
- `clicks`
- `spend`
- `conversions`
- `conversions_value`

过滤条件:

- `platform`
- `platform_account_id`
- `date_from`
- `date_to`
- `match_type`
- `search_term` 模糊查询
- `limit`

逻辑:

- 直接查 `olap_search_terms FINAL`
- 搜索词模糊匹配使用 `positionCaseInsensitiveUTF8(search_term, ?) > 0`
- 排序规则:
  - `stat_date DESC`
  - `conversions_value DESC`
  - `clicks DESC`
  - `search_term`

### 4.2.7 `/api/bi/game-kpis`

来源: `serving mysql.bi_game_kpis`

字段解释补充:

- 这部分不是广告平台直接返回的数据，而是业务/游戏侧补充写入的数据。
- 主要给 UA 报表拼接后链路使用。

返回字段就是 `GameKPIRecord` 全量字段:

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `platform_ad_group_id`
- `platform_ad_id`
- `stat_date`
- `country`
- `os`
- `placement`
- `creative_id`
- `creative_type`
- `optimization_goal`
- `bid_type`
- `targeting`
- `installs`
- `activations`
- `registrations`
- `tutorial_completions`
- `role_creations`
- `level_x_users`
- `purchasers`
- `purchase_count`
- `first_purchase_amount`
- `revenue_d1`
- `revenue_d7`
- `revenue_d30`
- `ad_revenue`
- `total_revenue`
- `retention_d1`
- `retention_d3`
- `retention_d7`
- `retention_d30`
- `ltv_d7`
- `ltv_d30`
- `avg_online_duration_seconds`
- `task_completion_rate`
- `high_value_payer_ratio`
- `raw_payload`

过滤条件:

- `platform`
- `platform_account_id`
- `date_from/date_to`
- `country`
- `os`
- `platform_campaign_id`
- `platform_ad_group_id`
- `platform_ad_id`
- `limit`

### 4.2.8 `/api/bi/ua-report`

来源:

- 广告侧: `clickhouse.olap_insights`
- 游戏侧: `serving mysql.bi_game_kpis`
- 合并逻辑: `internal/modules/reporting/application/ua_report_service.go`

字段解释补充:

- 这是“广告投放表现 + 游戏内转化/留存/收入”的合并报表。
- 广告侧字段偏媒体表现，游戏侧字段偏用户质量和变现质量。
- 派生字段如 `cpi/arpu/roi` 都是服务层实时计算，不单独落库。

#### 广告侧先查出的字段

`QueryUAReportRows()` 会从 `olap_insights` 聚合出:

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `entity_level`
- `entity_id`
- `platform_ad_group_id`
- `platform_ad_id`
- `stat_date`
- `device`
- `network`
- `impressions`
- `clicks`
- `spend`
- `ctr`
- `cpc`
- `cpm`
- `conversions`
- `all_conversions`
- `conversions_value`
- `cost_per_conversion`
- `cost_per_all_conversions`
- `reach`
- `frequency`
- `roas`

聚合维度:

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `entity_level`
- `entity_id`
- `platform_ad_group_id`
- `platform_ad_id`
- `stat_date`
- `device`
- `network`

派生逻辑:

- `ctr = clicks / impressions * 100`
- `cpc = spend / clicks`
- `cpm = spend / impressions * 1000`
- `cost_per_conversion = spend / conversions`
- `cost_per_all_conversions = spend / all_conversions`
- `frequency = impressions / reach`
- `roas = conversions_value / spend`

#### 游戏侧再补字段

`UAReportService.QueryReport()` 再查 `bi_game_kpis`，把这些字段按 key 合并进去:

- `country`
- `os`
- `placement`
- `creative_id`
- `creative_type`
- `optimization_goal`
- `bid_type`
- `targeting`
- `installs`
- `activations`
- `registrations`
- `tutorial_completions`
- `role_creations`
- `level_x_users`
- `purchasers`
- `purchase_count`
- `first_purchase_amount`
- `revenue_d1`
- `revenue_d7`
- `revenue_d30`
- `ad_revenue`
- `total_revenue`
- `retention_d1`
- `retention_d3`
- `retention_d7`
- `retention_d30`
- `ltv_d7`
- `ltv_d30`
- `avg_online_duration_seconds`
- `task_completion_rate`
- `high_value_payer_ratio`

合并 key:

- `stat_date`
- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `platform_ad_group_id`
- `platform_ad_id`
- `entity_level`
- `entity_id`
- `country`
- `os`

这意味着:

- 广告侧数据和游戏侧数据不是简单按“账号+日期”拼，而是按更细的对象粒度拼。
- 游戏 KPI 有 `country/os` 时，会参与 key；广告侧默认没有这两个维度。

#### `/api/bi/ua-report` 最终返回字段

最终返回 `UAReportRow`，包含:

- 广告侧原始聚合字段
- 游戏侧补充字段
- 以及下面这些服务层派生字段

服务层派生字段计算逻辑:

- `cpi = spend / installs`
- `activation_rate = activations / installs`
- `cpr = spend / registrations`
- `registration_rate = registrations / installs`
- `payer_rate = purchasers / activations`
- `arpu = total_revenue / activations`
- `arppu = total_revenue / purchasers`
- `roi = (total_revenue - spend) / spend`
- 如果 `total_revenue` 为空但 `revenue_d30 > 0`:
  - `roi` 改用 `(revenue_d30 - spend) / spend`
  - `total_revenue` 也会补成 `revenue_d30`
- 如果 `ltv_d30` 为空且 `installs > 0`:
  - `ltv_d30 = total_revenue / installs`
- `ltv_to_cpi_ratio = ltv_d30 / cpi`

### 4.2.9 `/api/bi/ua-fields`

这是字段目录接口，数据不是查库，而是 `server.go` 里硬编码的 catalog，用来告诉前端:

- 字段 key
- 中文 label
- category
- status
- source
- notes
- example_api
- related_keys

## 4.3 `/bi` 页面当前展示哪些字段

`/bi` 页面本质上是把多个 BI 接口查询结果拼到一起，再计算一批页面指标。

### 页面原始数据块

页面会同时查:

- `snapshots`
- `campaigns`
- `insight summary`
- `insight detail`
- `campaign diagnostics`
- `search terms`

### 页面顶层指标

`buildBIPageView()` 会计算:

- `SnapshotCount`
- `CampaignCount`
- `InsightRowCount`
- `InsightDetailRowCount`
- `CampaignDiagnosticRowCount`
- `SearchTermRowCount`
- `TotalImpressions`
- `TotalClicks`
- `TotalSpend`
- `TotalConversions`
- `TotalAllConversions`
- `TotalConversionsValue`
- `AvgCostPerConversion`
- `AvgCostPerAllConversions`
- `AvgSearchImpressionShare`
- `AvgSearchTopImpressionShare`
- `AvgSearchAbsoluteTopImpressionShare`
- `SearchTermClicks`
- `SearchTermSpend`
- `SearchTermConversions`
- `SearchTermConversionsValue`

这些都是页面内二次聚合结果，不是单独落库字段。

#### 页面聚合逻辑

- `Total*` 系列:
  - 基于 `insights summary` 汇总
- `AvgCostPerConversion`:
  - `TotalSpend / TotalConversions`
- `AvgCostPerAllConversions`:
  - `TotalSpend / TotalAllConversions`
- `AvgSearch*Share`:
  - 对 diagnostics 行做简单平均
- `SearchTerm*`:
  - 基于 `search terms` 汇总

### 页面图表和汇总

页面还有两组展示型衍生数据:

#### 平台汇总 `PlatformSummary`

数据来源:

- 账号/实体数量来自 `snapshots`
- 曝光/点击/花费来自 `insight summary`

字段:

- `platform`
- `account_count`
- `campaign_count`
- `ad_group_count`
- `ad_count`
- `insight_count`
- `total_impressions`
- `total_clicks`
- `total_spend`

#### 账号汇总 `AccountSummary`

同样是:

- 基础账号信息和 count 来自 `snapshots`
- 曝光/点击/花费来自 `insight summary`

字段:

- `platform`
- `account_id`
- `account_name`
- `source_mode`
- `campaign_count`
- `ad_group_count`
- `ad_count`
- `insight_count`
- `total_impressions`
- `total_clicks`
- `total_spend`

#### 趋势图

`ImpressionsSVG` / `ClicksSVG` / `SpendSVG` 的逻辑:

- 先对 `insight summary` 按 `stat_date` 汇总
- 再渲染成 SVG 柱状图

这部分也是页面展示逻辑，不是新存储字段。

## 5. 当前实现里的几个重要事实

### 5.1 raw 和 trans/bi 不是一一等宽传递

当前链路不是“raw 有什么字段，bi 就都能查到什么字段”。

实际情况是:

- raw `payload_json` 可以保留很多原始字段
- trans 只挑 normalizer 里显式解码的字段
- BI 只暴露 read model / query model 里定义过的字段

所以如果某个字段已经进了 raw，但 normalizer 没接、projector 没写、BI query 没查，那么 BI 侧仍然不可见。

### 5.2 当前 account 维度是弱实现

虽然代码里有:

- `ObjectTypeAccount`
- `StandardAccount`
- `FetchAccounts`

但当前主链路里:

- account 维度主要依赖 `SyncTarget.Bundle.Account`
- trans 没有完整的 account normalizer + projector 路径
- BI 也没有独立 account 详情表接口

所以账户信息现在更像“同步目标维度补充信息”，而不是完整 raw account 标准化产物。

### 5.3 Google Ads 的字段覆盖最完整

从当前代码看:

- Google Ads 已经接了 `campaign diagnostics`
- Google Ads 已经接了 `search terms`
- Google Ads 的 insight 字段最全，包含 `all_conversions / conversions_value / cost_per_*`

Facebook 和 TikTok 当前只接了更基础的 campaign/adgroup/ad/insight 字段。

### 5.4 BI 有两类字段

第一类是“存储字段”:

- 来自 MySQL / ClickHouse 表

第二类是“展示派生字段”:

- `/api/bi/ua-report` 服务层计算出的 `cpi/roi/arpu/...`
- `/bi` 页面里算出的 total/avg/trend/platform summary/account summary

后者并没有单独落库存储。

## 6. 建议后续怎么继续维护这份链路

如果后面要继续扩字段，建议按这条顺序检查:

1. connector 是否已经把字段放进 raw payload
2. normalizer 是否已经把字段解到标准模型
3. projector 是否已经把字段写入 serving mysql / clickhouse
4. BI repository 是否已经查出这个字段
5. API / 页面是否已经返回和展示这个字段

只改其中一层，通常还不够。
