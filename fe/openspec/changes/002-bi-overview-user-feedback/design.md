# Design: BI Overview Phase 1 UI Alignment

## 当前数据面

React 端通过 `loadBIDashboard(filters)` 一次性加载 BI 数据，主要接口是：

- `/api/bi/insights/summary`：广告侧按天汇总。
- `/api/bi/insights/detail`：广告侧实体、设备、network 明细。
- `/api/bi/ua-report`：广告侧 + 游戏侧合并后的 UA 报告。
- `/api/bi/game-kpis`：游戏侧 KPI。
- `/api/bi/snapshots`、`/api/bi/campaigns`：账号和 campaign 结构数据。

当前系统里的 `platform` 实际含义是广告来源/渠道，和用户反馈里的 `media_source` 更接近。Phase 1 只改 UI 文案和前端展示，不改后端 query key、领域模型和库表字段。

## 目标 Overview 结构

```text
Overview
  日期范围
    start_date
    end_date

  业务筛选
    media_source
    region/country
    device/platform
    campaign
    ad_group
    ad
    product: Phase 2 接入后加入

  KPI 卡片
    spend
    installs
    CPI
    ROAS
    purchase
    CPA: 待确认口径
    D1 retention
    CPM
    CTR
    CVR: 待确认口径
    D0 LTV: Phase 2
    D7 LTV

  按天趋势
    支持切换已具备数据的指标

  汇总表
    默认维度: media_source, country/region
    可选维度: device/platform, campaign, ad_group, ad, account
    product: Phase 2 接入后成为可选维度
    指标列: 与 KPI 卡片同一套口径
```

## 筛选区设计

Overview 主筛选保留：

- Media Source：映射当前 `platform`。
- Region/Country：映射当前 `country`，但要注意广告侧现在没有完整地域字段。
- Device/Platform：先映射当前 `device` / `os`，等“平台”语义确认后再定名。
- Campaign、Ad Group、Ad：映射当前 `platform_campaign_id`、`platform_ad_group_id`、`platform_ad_id`。
- Start Date、End Date：映射当前 `date_from`、`date_to`。

Overview 主筛选移除或弱化：

- Product：Phase 1 不做假筛选；Phase 2 确认来源后接入。
- Account：默认隐藏，可作为高级筛选或可选维度。
- Entity Level：如果选了 campaign/ad_group/ad，可以由选择层级推断。
- Network：偏广告平台诊断，放到 Breakdown。
- Match Type、Search Term：放到 Breakdown/Search Terms。
- Detail Limit、Search Term Limit：保留为内部默认参数，不在 Overview 主区展示。

## 指标计算

Phase 1 需要抽一层统一的前端聚合 helper，让 KPI 卡片、按天趋势、汇总表使用同一套口径，避免同一页面上数字对不上。

| 指标 | Phase 1 公式或展示 |
| --- | --- |
| spend | 汇总广告 spend |
| installs | 汇总 installs；无游戏 KPI 时显示未接入 |
| CPI | spend / installs |
| ROAS | 使用当前 UA row `roas`，标注不是 D0/D7 口径 |
| purchase | 暂用 `purchasers` 或 `purchase_count` 中被确认的一项；未确认则空态 |
| CPA | 口径未确认，先空态或待确认 |
| D1 retention | 使用 `retention_d1` |
| CPM | spend * 1000 / impressions |
| CTR | clicks / impressions |
| CVR | 口径未确认，先空态或待确认 |
| D0 LTV | Phase 2 接入，Phase 1 不展示假值 |
| D7 LTV | 使用 `ltv_d7`；无游戏 KPI 时显示未接入 |

比率类指标建议尽量用“汇总后的分子 / 汇总后的分母”，不要简单平均每行 ratio。这样按 media_source、region 汇总时更稳定。

## 汇总表设计

Overview 表格改成类似 pivot 的汇总视角：

- 维度是可选列。
- 指标是数值列。
- 默认维度是 media_source + country/region。
- account 默认不展示，只作为可选维度或高级筛选。
- account/campaign/ad 数量这类库存计数默认不放 Overview，除非用户主动选择。

当前 UA 明细表更适合放在 Breakdown 或 Quality，用于排查数据完整性、定位 campaign/ad_group/ad 明细。

## 空态和提示

- 游戏侧为空时，install、purchase、retention、LTV 类指标显示“游戏侧未接入”。
- D0 LTV、D0 ROAS、D7 ROAS 在 Phase 1 不可用时显示“待 Phase 2 接入”，不展示 0 误导用户。
- CPA、CVR 未确认口径时显示“口径待确认”，避免和广告平台口径混淆。

## Phase 2 数据缺口

Product 当前没有建模。要做真实筛选，需要先决定来源，然后再贯通前端类型、后端 filter、BI read model、存储和数据写入。

广告侧 ClickHouse `olap_insights` 当前没有 country/OS。country/OS 主要来自游戏侧 `bi_game_kpis` 和合并后的 UA row，所以“分地区汇总”在广告侧数据不完整时可能会出现空值或只覆盖游戏侧行。

D0 LTV、D0 ROAS、D7 ROAS 当前没有明确字段或口径。Phase 1 不用 D1 或 total revenue 冒充，Phase 2 通过 Spec Kit 补真实链路。
