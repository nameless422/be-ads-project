# Tasks: BI Overview Phase 1 UI Alignment

- [ ] 将当前广告来源筛选从 `Platform` 展示为 `Media Source / media_source`，后端仍使用 `platform` query key。
- [ ] 将 Overview 筛选区拆成日期范围和业务层级两组。
- [ ] 从 Overview 主筛选区移走 network、match type、search term、detail limit、search term limit 控件。
- [ ] 默认隐藏 account，保留为高级筛选或可选维度。
- [ ] 将当前 10 张 KPI 卡片替换为用户指定的核心指标集合。
- [ ] 对游戏侧未接入的 installs、purchase、retention、LTV 指标展示清晰空态。
- [ ] 对 D0 LTV、D0 ROAS、D7 ROAS 展示 Phase 2 待接入状态，不用其它指标冒充。
- [ ] 对 CPA、CVR 展示口径待确认状态，或在业务口径确认后再启用。
- [ ] 增加按天趋势模块，支持在已具备数据的指标之间切换。
- [ ] 用可选维度 + 指标列的汇总表替换当前 Platform Summary、Account Snapshots、UA Overview。
- [ ] 将 UA 明细表入口移动到 Breakdown 或 Quality，而不是默认堆在 Overview。
- [ ] 执行 `npm run build` 验证前端构建。
