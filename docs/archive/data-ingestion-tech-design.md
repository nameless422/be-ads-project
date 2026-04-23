# 广告投放系统技术方案

这是一份归档文档，用来保留早期 ingestion 方案草案，不作为当前实现和目录结构的直接依据。

## 1. 目标范围

第一阶段先完成数据采集层（Data Ingestion），目标是把多广告平台的数据稳定拉回到内部系统，并沉淀统一标准数据，优先支持：

- Facebook Marketing API
- Google Ads API
- TikTok Ads API（预留接入能力）

本阶段重点能力：

- 定时拉取
- 增量同步
- 多账户配置
- 字段标准化
- 基础可观测性与失败重试

暂不重点覆盖：

- BI 分析报表
- 自动投放优化
- 复杂归因建模
- 实时流式消费

## 2. 总体架构

建议采用“控制面 + 采集执行面 + 标准化存储面”三层架构。

```text
+-----------------------+
|   Admin / Config UI   |
+-----------+-----------+
            |
            v
+-----------------------+
| Account Config Center |
| - 平台账号配置         |
| - 授权信息管理         |
| - 拉取策略配置         |
+-----------+-----------+
            |
            v
+-----------------------+        +----------------------+
|  Scheduler            | -----> |   Job Queue          |
| - cron / Airflow      |        | - sync jobs          |
| - 生成拉取任务         |        | - retry / backoff    |
+-----------+-----------+        +----------+-----------+
            |                               |
            v                               v
+-----------------------+        +----------------------+
| Connector Workers     | -----> | Raw Data Storage     |
| - Facebook            |        | - 原始响应           |
| - Google Ads          |        | - 请求游标/水位      |
| - TikTok              |        +----------+-----------+
+-----------+-----------+                   |
            |                               v
            +---------------------> +----------------------+
                                    | Standardized Tables  |
                                    | - account/campaign   |
                                    | - adset/ad/ad insight|
                                    +----------------------+
```

## 3. 技术选型建议

考虑你当前目录是 Go 项目，第一阶段建议主服务直接用 Go 实现，避免调度、采集、标准化拆得太散。

推荐组合：

- 服务语言：Go
- 调度：
  - 简版先用服务内 cron
  - 后续任务规模上来再切 Airflow/外部调度
- 队列：
  - MVP 先用数据库任务表
  - 后续可替换 Redis / RabbitMQ / Kafka
- 存储：
  - MySQL / PostgreSQL 存业务与标准化数据
  - 对象存储或原始表保存 raw payload
- 配置与密钥：
  - 数据库存非敏感配置
  - 密钥类信息放 KMS / Secret Manager / 加密字段

第一阶段建议不要一上来引入 Airflow。原因是当前核心问题是“先把多平台、多账户、增量同步模型做对”，不是先把编排系统做重。

## 4. 模块拆分

建议代码结构如下：

```text
be_ads_project/
  cmd/
    server/
    worker/
  internal/
    ingestion/
      scheduler/
      dispatcher/
      job/
      service/
    connector/
      meta/
      facebook/
      googleads/
      tiktok/
    normalize/
    repository/
    domain/
    encrypt/
  docs/
```

模块职责：

- `scheduler`
  - 扫描启用中的账户与同步配置
  - 按粒度生成同步任务
- `dispatcher`
  - 控制并发、重试、限流
- `connector/*`
  - 封装各平台 API 拉取逻辑
  - 处理分页、鉴权、平台侧限流
- `normalize`
  - 将平台返回数据映射为统一字段
- `repository`
  - 操作配置表、任务表、原始表、标准化表
- `domain`
  - 定义统一实体与枚举

## 5. 数据对象优先级

第一阶段建议不要一次性把所有对象都采完，先做最核心且跨平台最容易统一的一层。

优先顺序：

1. 广告账户 Account
2. Campaign
3. Ad Set / Ad Group
4. Ad
5. Insight / Report Metrics

原因：

- 账户、Campaign、Ad Group、Ad 的结构性对象比较稳定
- 指标数据依赖时间窗与归因口径，复杂度更高，但业务价值最高
- 先把对象主链路打通，再接日报/小时级指标同步更稳

## 6. 多账户配置设计

核心思路：平台账户、授权凭证、同步策略解耦。

### 6.1 关键表设计

#### `platform_accounts`

记录一个“可拉取的广告账户”。

字段建议：

- `id`
- `platform`：`facebook` / `google_ads` / `tiktok_ads`
- `account_id`：平台原始账户 ID
- `account_name`
- `status`：`active` / `disabled` / `auth_expired`
- `timezone`
- `currency`
- `business_id`：归属业务线或租户
- `created_at`
- `updated_at`

#### `platform_account_credentials`

记录授权与敏感信息，建议单独隔离。

字段建议：

- `id`
- `platform_account_id`
- `auth_type`
- `access_token`
- `refresh_token`
- `token_expire_at`
- `app_id` / `client_id`
- `app_secret` / `client_secret`
- `extra_config`：JSON，存 developer token、customer id 等平台特有参数
- `created_at`
- `updated_at`

建议：

- 敏感字段加密存储
- Connector 层只拿解密后的运行时配置

#### `sync_profiles`

定义“这个账户要同步什么、多久同步一次、从哪里开始增量”。

字段建议：

- `id`
- `platform_account_id`
- `object_type`：`account` / `campaign` / `ad_group` / `ad` / `insight`
- `sync_mode`：`full_bootstrap` / `incremental`
- `schedule_type`：`cron`
- `schedule_expr`
- `lookback_window_minutes`
- `watermark_field`
- `watermark_value`
- `page_token`
- `is_enabled`
- `last_success_at`
- `last_error_at`
- `updated_at`

这个设计能做到：

- 同一个账户可以对不同对象配置不同同步周期
- 首次全量与后续增量可以复用同一套任务引擎
- TikTok 接进来时不用改主框架

## 7. 定时拉取设计

第一阶段建议采用“两级调度”：

- 一级：定时扫描 `sync_profiles`
- 二级：为每个 profile 生成具体执行任务

任务表示例：`sync_jobs`

字段建议：

- `id`
- `platform_account_id`
- `sync_profile_id`
- `platform`
- `object_type`
- `job_type`：`bootstrap` / `incremental`
- `job_status`：`pending` / `running` / `success` / `failed`
- `scheduled_at`
- `started_at`
- `finished_at`
- `retry_count`
- `request_cursor`
- `watermark_snapshot`
- `error_message`

### 7.1 调度策略

建议初期频率：

- 账户/Campaign/Ad Group/Ad：每 15-30 分钟
- Insight 指标：每 1-6 小时

原因：

- 结构对象变化不会特别频繁
- 指标数据会有延迟回补，频率过高收益不大

### 7.2 执行方式

每次任务执行流程：

1. 读取账户配置与凭证
2. 加载 `sync_profile`
3. 构造 connector 请求
4. 按分页游标拉取
5. 原始响应落盘/落表
6. 标准化转换
7. Upsert 到标准化表
8. 更新 watermark / page token / last_success_at

## 8. 增量同步设计

这是第一阶段最关键的部分。

### 8.1 统一增量模型

不要把“增量”只理解成 `updated_at > 上次时间`。不同平台支持能力不一致，所以要抽象成统一的 checkpoint。

统一 checkpoint 建议由三部分组成：

- `time_watermark`
- `page_cursor`
- `lookback_window`

说明：

- `time_watermark`
  - 表示上次同步到哪个业务时间点
- `page_cursor`
  - 解决单次窗口内分页未拉完的问题
- `lookback_window`
  - 解决平台数据延迟、回写、归因修正

### 8.2 推荐增量策略

#### 结构型对象

如 `campaign/adgroup/ad`：

- 优先用平台支持的更新时间过滤
- 如果平台不稳定，采用“最近 N 天回扫 + 主键 upsert”

推荐默认：

- 每次拉最近 3-7 天更新数据
- 以平台主键做幂等更新

这样虽然有少量重复拉取，但比严格依赖单一 watermark 更稳。

#### 指标型对象

如 `insight`：

- 按时间粒度分段拉取
- 最近 3 天高频回刷
- 最近 7-30 天低频补刷

原因：

- 广告平台常出现指标延迟归因
- 当天、昨天数据不稳定

推荐策略：

- `T-0 ~ T-2`：每小时刷新
- `T-3 ~ T-7`：每天刷新
- `T-8 ~ T-30`：按需补数

### 8.3 幂等与去重

统一要求：

- 原始拉取可重复
- 标准化写入必须幂等

实现方式：

- 结构对象：唯一键 `platform + account_id + object_id`
- 指标对象：唯一键 `platform + account_id + object_id + stat_date + breakdown`
- 使用 `upsert`

## 9. 字段标准化设计

必须在 connector 与业务查询之间加一层标准模型，不然后面接第三个平台会很痛。

### 9.1 标准字段示例

#### 标准化 `campaign`

- `platform`
- `platform_account_id`
- `platform_campaign_id`
- `campaign_name`
- `status`
- `objective`
- `buying_type`
- `daily_budget`
- `lifetime_budget`
- `currency`
- `start_time`
- `end_time`
- `updated_at`
- `raw_payload`

#### 标准化 `insight`

- `platform`
- `platform_account_id`
- `entity_level`
- `entity_id`
- `stat_date`
- `impressions`
- `clicks`
- `spend`
- `ctr`
- `cpc`
- `cpm`
- `conversions`
- `reach`
- `raw_payload`

### 9.2 字段映射方式

建议每个平台单独维护 mapper：

- `facebook/campaign_mapper.go`
- `googleads/campaign_mapper.go`
- `tiktok/campaign_mapper.go`

统一输出到：

- `domain.StandardCampaign`
- `domain.StandardInsight`

这样业务层永远依赖标准结构，不直接感知 Facebook/Google 的字段差异。

## 10. Connector 抽象设计

建议定义统一接口：

```go
type Connector interface {
    Platform() string
    Validate(ctx context.Context, cfg AccountCredential) error
    FetchAccounts(ctx context.Context, req FetchRequest) (*FetchResult, error)
    FetchCampaigns(ctx context.Context, req FetchRequest) (*FetchResult, error)
    FetchAdGroups(ctx context.Context, req FetchRequest) (*FetchResult, error)
    FetchAds(ctx context.Context, req FetchRequest) (*FetchResult, error)
    FetchInsights(ctx context.Context, req FetchRequest) (*FetchResult, error)
}
```

其中：

- `FetchRequest` 统一封装时间窗、分页、账户配置、字段列表
- `FetchResult` 返回：
  - 标准原始记录切片
  - `nextPageToken`
  - `nextWatermark`
  - `hasMore`

这样 TikTok 后续接入时，只是新增一个 connector 实现。

## 11. 限流、重试与容错

广告平台 API 很容易遇到限流、临时失败、授权过期。

第一阶段必须具备：

- 指数退避重试
- 平台级并发控制
- 单账户级串行或低并发
- 鉴权失败自动标记账户异常

建议：

- 平台维度配置 QPS / 并发上限
- 429 / 5xx 可重试
- 401 / 403 标记 `auth_expired`
- 单个账户失败不影响其他账户

## 12. 可观测性

最少要有：

- job 维度日志
- 任务状态表
- 成功/失败计数
- 平台 API 耗时
- 每次拉取记录数

建议指标：

- `sync_job_total`
- `sync_job_failed_total`
- `sync_records_fetched_total`
- `sync_records_upserted_total`
- `connector_api_latency_ms`

## 13. 第一阶段落地顺序

建议分 4 个小阶段做，不要同时啃完所有平台所有对象。

### Phase 1

- 多账户配置表
- 凭证管理
- 同步任务表
- 基础 cron scheduler
- Facebook Connector 骨架

### Phase 2

- Facebook Campaign / AdSet / Ad 拉取
- 原始数据存储
- 标准化表与 upsert
- watermark 更新

### Phase 3

- Google Ads Connector 接入
- 统一 mapper 与平台差异收敛
- 限流与重试增强

### Phase 4

- Insight 指标同步
- 回刷补数策略
- TikTok Connector 预接入

## 14. MVP 推荐范围

如果你想尽快做出第一版可用系统，我建议 MVP 只做：

- 平台：Facebook
- 对象：Campaign / AdSet / Ad
- 能力：多账户配置 + 定时拉取 + 增量同步 + 标准化入库

先不把 Insight 放进 MVP 的原因：

- 指标口径复杂
- 数据回补多
- 不同平台差异大

先把对象主链路做通，后面加指标会轻很多。

## 15. 风险点

需要提前注意：

- 不同平台授权方式差异大
- Google Ads API 学习成本高于 Facebook
- 指标定义与归因口径不统一
- 只依赖 watermark 可能漏数
- 多账户并发容易触发平台限流

所以第一阶段的设计原则应是：

- 允许重复拉取，不允许漏数
- 先统一采集框架，再扩平台
- 先保证幂等，再追求极致性能

## 16. 推荐结论

如果目标是“先把采集和多账户配置做好”，建议路线是：

1. 先用 Go 做单体采集服务
2. 用数据库任务表承接调度与执行状态
3. 先接 Facebook，沉淀统一 connector 接口
4. 用 `sync_profiles + watermark + lookback_window` 做增量模型
5. 标准化对象先做 `campaign/adgroup/ad`
6. 所有写入一律按唯一键 upsert
7. TikTok 与 Google Ads 后续作为 connector 扩展接入

这个路线最稳，也最适合后续继续扩成完整广告数据中台。

## 17. 下一步建议

如果继续往下做实现，建议下一步直接产出下面三样：

1. 数据库表结构 SQL
2. Go 项目目录骨架
3. Facebook Connector 接口与任务调度最小实现

这样方案就能从“文档”进入“可编码状态”。
