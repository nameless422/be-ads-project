# be-ads-project

广告数据采集、转换和 BI 展示的本地工程。

当前仓库已经拆成两个主目录：

```text
be/  Go 后端、采集/转换/BI API、本地运维脚本、后端 SDD
fe/  React + Vite BI 前端、前端 SDD
```

根目录只保留项目入口，具体代码和文档都在 `be/`、`fe/` 内。

## 能做什么

- 从广告平台账号维度组织采集任务
- 写入 raw 数据和 outbox 事件
- 将 raw event 转换为标准 BI read model
- 投影到 MySQL / ClickHouse
- 通过 `/api/bi/*` 提供 BI 查询接口
- 通过 `/bi` 提供 React BI 页面
- 通过 `/api/control/*` 和控制页管理本地栈

## 快速启动

### Mac 新环境

首次 clone 后，推荐直接跑一键启动：

```bash
git clone git@github.com:nameless422/be-ads-project.git
cd be-ads-project/be
make mac-start
```

这个命令会检查并准备本地依赖：

- Xcode Command Line Tools
- Homebrew
- Go
- Node.js / npm
- Docker Desktop
- 本地端口

然后会安装前端依赖、启动 MySQL / ClickHouse / NATS / 后端服务，并执行本地主链路验收。

启动成功后访问：

```text
http://127.0.0.1:8080/
http://127.0.0.1:8080/bi
```

### 已有环境

```bash
cd be
make up
make verify
make status
make down
```

## 后端

后端入口在 `be/`：

```bash
cd be
make help
make test
```

主要目录：

```text
cmd/       bi-api、collector-worker、transformer-worker、control-plane
internal/ 业务模块和基础设施适配
scripts/  本地启动、停止、验证脚本
deploy/   部署和观测配置
```

更多说明见 [be/README.md](be/README.md)。

## 前端

前端入口在 `fe/`：

```bash
cd fe
npm install
npm run dev
npm run build
```

生产环境由后端 `bi-api` 从 `fe/dist` 托管 `/bi` 页面。

更多说明见 [fe/README.md](fe/README.md)。

## SDD

项目同时保留两套 SDD 方式：

```text
be/openspec   后端轻量 OpenSpec
be/specs      后端 Spec Kit
fe/openspec   前端轻量 OpenSpec
fe/specs      前端 Spec Kit
```

建议：

- 小改动走 OpenSpec
- 大重构、跨前后端、API 或数据链路变化走 Spec Kit
- 临时问题和一次性排查可以直接对话处理

## 常用入口

```bash
cd be
make mac-start          # Mac 新环境一键启动
make up                 # 启动本地栈
make verify             # 验收本地主链路
make status             # 查看服务状态
make down               # 停止本地栈
make test               # 后端测试
make frontend-build     # 构建前端
```

## 当前边界

这个仓库优先服务本地开发和链路验证。真实平台账号、生产部署、安全权限、完整 BI 产品化能力需要在对应 spec 中单独展开。
