# Google Ads Test Account 接入与切换 Runbook

## 目标

这份文档解决两件事：

1. 用现有的 Google Ads `test manager + 5 个 test client` 验证采集链路。
2. 为后续切到真实 Google Ads 账号提前做好配置和执行步骤。

当前仓库已经支持两种模式：

- `seeded_test`
  账号 ID 使用真实 test account 的 `customer_id`，但 payload 是本地生成的测试数据。
- `real`
  通过 Google Ads API 真实拉取 `campaign / ad_group / ad / insight`。

## 已有 Google Ads 资源

### Manager 与 Developer Token

- `production manager account`: `284-020-6764`
- `developer token`: `yOCuBAsqVI0UyT94T7jDow`
- `developer token access level`: `Test Account`

### Test Hierarchy

- `test manager account`: `357-594-0005`
- `test client 01`: `248-390-1805`
- `test client 02`: `492-825-2952`
- `test client 03`: `691-332-4649`
- `test client 04`: `400-404-3492`
- `test client 05`: `608-174-8445`

### OAuth / Cloud Project

- `google cloud project`: `be-ads-project`
- `google ads api`: 已启用
- `oauth consent audience`: `External`
- `oauth publish status`: `Testing`
- `oauth test user`: `873190934@qq.com`
- `oauth desktop client`: 已创建

注意：

- `Testing` 状态下，只有加到 test users 的账号可以授权
- 当前仓库已经提供本地 OAuth helper：
  - [`cmd/google_oauth_bootstrap/main.go`](/Users/zhongyi.zhang/project/go/be_ads_project/cmd/google_oauth_bootstrap/main.go)

## Google 官方限制

根据 Google 官方文档：

- test account 不投放真实广告，也不会产生真实展示和花费
- serving metrics 可能为空
- test hierarchy 不能和 production hierarchy 混用
- 单个 test hierarchy 最多 50 个账号

参考：

- [Test accounts](https://developers.google.com/google-ads/api/docs/best-practices/test-accounts)
- [Testing best practices](https://developers.google.com/google-ads/api/docs/best-practices/testing)
- [Authorization and HTTP headers](https://developers.google.com/google-ads/api/rest/auth)
- [OAuth overview](https://developers.google.com/google-ads/api/docs/oauth/overview)

## 当前代码状态

### 已完成

- 已把 5 个 test client account 接进系统启动配置
- 已增加 Google Ads `real / seeded_test` 模式切换
- 已支持真实模式下通过 REST `searchStream` 拉：
  - `campaign`
  - `ad_group`
  - `ad`
  - `insight`
- 已把 `raw_record` 和 `normalized_record` 打进日志，方便 `tail -f`

### 当前默认行为

如果缺少 OAuth 凭证，系统默认回退到：

```text
BE_GOOGLE_ADS_MODE=seeded_test
```

此时日志会出现：

```text
source_mode=seeded_test
```

这表示：

- 账号是你真实创建的 test account
- 但采集内容仍然是本地 seeded payload
- 目的是先验证调度、标准化、日志观测链路

## 当前真实接入状态

这一步已经完成：

- 已拿到 `client_id`
- 已新增可用的 `client_secret`
- 已通过浏览器授权拿到 `refresh_token`
- 已把服务切到 `BE_GOOGLE_ADS_MODE=real`
- 已对 Google Ads API 发起真实拉取

已验证到的日志特征：

```text
source_mode=real
```

以及：

```text
platform=google_ads account=248-390-1805 ...
platform=google_ads account=492-825-2952 ...
```

当前真实返回结果为：

- `campaign=0`
- `ad_group=0`
- `ad=0`
- `insight=0`

这说明两件事：

1. OAuth、developer token、login_customer_id、customer_id 这一整条真实链路已经通了。
2. 目前 test client 里还没有可被这几条 GAQL 查到的对象，或者对象还没创建完整。

## 浏览器内需要完成的前置操作

### 1. 创建一个 Google Cloud Project

建议项目名：

- `be-ads-project`

用途：

- 开启 `Google Ads API`
- 创建 OAuth client
- 配置 OAuth consent screen

### 2. 启用 Google Ads API

在 Google Cloud Console 中进入：

- `API 和服務`
- `程式庫`
- 搜索 `Google Ads API`
- 点击 `Enable`

### 3. 配置 OAuth Consent Screen

建议最小配置：

- App type: `External`
- App name: `be_ads_project`
- Support email: 你自己的邮箱
- Developer contact email: 你自己的邮箱

如果只是自己调试，初期保持最小配置即可。

### 4. 创建 OAuth Client

推荐先创建：

- `Desktop app`

原因：

- 最容易拿到一次性的授权码和 `refresh_token`
- 适合本地开发调试

创建完成后会得到：

- `client_id`
- `client_secret`

### 5. 生成 Refresh Token

授权 scope 使用：

```text
https://www.googleapis.com/auth/adwords
```

拿 `refresh_token` 的方式有两种：

1. 用 OAuth Playground 或本地授权脚本完成一次授权
2. 在浏览器走 Desktop App 授权流程，换取 authorization code，再换取 refresh token

Google 官方说明见：

- [OAuth overview](https://developers.google.com/google-ads/api/docs/oauth/overview)
- [Authorization and HTTP headers](https://developers.google.com/google-ads/api/rest/auth)

### 6. 把授权账号加入 OAuth Test Users

如果 OAuth app 还是 `Testing` 状态，必须在 Google Auth Platform 的 `Audience` 页面里把实际授权账号加入 `Test users`。

否则浏览器授权会报：

```text
access_denied
The developer hasn’t given you access to this app
```

## 推荐的本地环境变量

参考文件：

- [.env.google-ads.example](/Users/zhongyi.zhang/project/go/be_ads_project/.env.google-ads.example)

真实模式所需配置：

```bash
export BE_GOOGLE_ADS_MODE=real
export BE_GOOGLE_ADS_DEVELOPER_TOKEN=yOCuBAsqVI0UyT94T7jDow
export BE_GOOGLE_ADS_LOGIN_CUSTOMER_ID=3575940005
export BE_GOOGLE_ADS_CLIENT_ID=your_client_id
export BE_GOOGLE_ADS_CLIENT_SECRET=your_client_secret
export BE_GOOGLE_ADS_REFRESH_TOKEN=your_refresh_token
export BE_GOOGLE_ADS_API_VERSION=v20
```

说明：

- `login_customer_id` 可以带横杠，也可以不带。代码里会自动清洗。
- `customer_id` 不需要手工逐个写，系统已内置 5 个 test client。
- `client_id / client_secret / refresh_token` 不要提交进仓库。

## 本地拿 Refresh Token 的命令

仓库里已经有一个 helper，可以直接拉起浏览器授权并在本地回调端口换取 token：

```bash
cd /Users/zhongyi.zhang/project/go/be_ads_project
BE_GOOGLE_ADS_CLIENT_ID=your_client_id \
BE_GOOGLE_ADS_CLIENT_SECRET=your_client_secret \
go run ./cmd/google_oauth_bootstrap
```

成功后会打印：

```text
export BE_GOOGLE_ADS_CLIENT_ID='...'
export BE_GOOGLE_ADS_CLIENT_SECRET='...'
export BE_GOOGLE_ADS_REFRESH_TOKEN='...'
```

## 如何运行

### seeded_test 模式

```bash
cd /Users/zhongyi.zhang/project/go/be_ads_project
./scripts/ops/start.sh
tail -f /Users/zhongyi.zhang/project/go/be_ads_project/logs/collector-worker.stdout.log
```

你会看到：

```text
[collector-worker] ... source_mode=seeded_test
[transformation] raw_record ...
[transformation] normalized_record ...
```

### real 模式

先导出环境变量，再启动：

```bash
cd /Users/zhongyi.zhang/project/go/be_ads_project
export BE_GOOGLE_ADS_MODE=real
export BE_GOOGLE_ADS_DEVELOPER_TOKEN=your_developer_token
export BE_GOOGLE_ADS_LOGIN_CUSTOMER_ID=3575940005
export BE_GOOGLE_ADS_CLIENT_ID=your_client_id
export BE_GOOGLE_ADS_CLIENT_SECRET=your_client_secret
export BE_GOOGLE_ADS_REFRESH_TOKEN=your_refresh_token
./scripts/ops/start.sh
tail -f /Users/zhongyi.zhang/project/go/be_ads_project/logs/collector-worker.stdout.log
```

如果真实模式生效，日志应出现：

```text
source_mode=real
```

如果真实模式生效，日志会出现类似：

```text
[collector-worker] ... source_mode=real
```

当前已实际验证：

```text
[transformation] normalized ... platform=google_ads ... raw=0 campaigns=0 adgroups=0 ads=0 insights=0
```

这不是鉴权失败，而是当前 test account 里还没有被查询到的数据。

## 真实模式下当前支持的拉取对象

### campaign

GAQL:

```sql
SELECT
  campaign.id,
  campaign.name,
  campaign.status,
  campaign.advertising_channel_type,
  campaign_budget.amount_micros
FROM campaign
ORDER BY campaign.id
LIMIT 50
```

### ad_group

GAQL:

```sql
SELECT
  ad_group.id,
  ad_group.name,
  ad_group.status,
  ad_group.cpc_bid_micros,
  campaign.id
FROM ad_group
ORDER BY ad_group.id
LIMIT 50
```

### ad

GAQL:

```sql
SELECT
  ad_group_ad.ad.id,
  ad_group_ad.ad.name,
  ad_group_ad.status,
  ad_group.id
FROM ad_group_ad
ORDER BY ad_group_ad.ad.id
LIMIT 50
```

### insight

GAQL:

```sql
SELECT
  campaign.id,
  segments.date,
  metrics.impressions,
  metrics.clicks,
  metrics.cost_micros,
  metrics.ctr,
  metrics.average_cpc,
  metrics.average_cpm,
  metrics.conversions
FROM campaign
WHERE segments.date DURING YESTERDAY
ORDER BY campaign.id
LIMIT 50
```

说明：

- 对 test account，`campaign / ad_group / ad` 更适合做结构验证
- `insight` 字段能否返回有效值，取决于 test account 的限制

## 切换到真实账号的准备

当 developer token 升级到可访问 production account 后，切换方式尽量保持不变：

### 需要替换的只有这几项

- `BE_GOOGLE_ADS_LOGIN_CUSTOMER_ID`
- `BE_GOOGLE_ADS_CLIENT_ID`
- `BE_GOOGLE_ADS_CLIENT_SECRET`
- `BE_GOOGLE_ADS_REFRESH_TOKEN`
- `internal/mock/data.go` 中内置的 test client account 列表

### 推荐做法

不要直接改代码里的账号清单，建议下一步把账号来源改成：

- 配置文件
- 数据库表
- 或环境变量注入

这样 test / prod 切换只需要改配置，不需要改编译产物。

## 你现在可以怎样验证

### 验证 1

确认真实模式已经启用：

- 看日志里是否出现 `source_mode=real`

### 验证 2

确认 test account 会被真实调度：

- 看日志里是否出现 5 个 `account=...`

### 验证 3

确认 Google 返回是否为空集合：

- 看 `normalized ... raw=0` 还是 `raw>0`

### 验证 4

如果还是 `raw=0`，去 test client 里手工建最小对象：

- 1 个 campaign
- 1 个 ad group
- 1 个 ad

然后重新启动服务再看日志。

## 当前结论

到目前为止，这条链路已经从“本地 mock”推进到了“真实 Google Ads API 拉取”：

- test hierarchy 已建好
- OAuth 已配置完成
- `refresh_token` 已拿到
- 代码已切到 `real` 模式并成功请求 Google Ads API
- 当前返回 0 条的原因，不是接入失败，而是 test client 里还没有查询到对象

下一步如果要看到非 0 数据，最直接的动作就是：

1. 在任意一个 test client 里手工创建 `campaign / ad group / ad`
2. 重跑服务
3. 再看 `raw_record / normalized_record`

## 切到真实生产账号时怎么做

当 developer token 具备 production access 后，切换动作尽量只改配置，不改代码：

1. 替换 `BE_GOOGLE_ADS_LOGIN_CUSTOMER_ID`
2. 替换 `BE_GOOGLE_ADS_CLIENT_ID`
3. 替换 `BE_GOOGLE_ADS_CLIENT_SECRET`
4. 替换 `BE_GOOGLE_ADS_REFRESH_TOKEN`
5. 把 `internal/mock/data.go` 里的 test account 清单替换成真实账号来源

更推荐的长期做法：

- 账号列表从数据库或配置文件加载
- OAuth 凭证走密钥管理或环境变量注入
- test / prod 只通过配置切换
