# fe

React + Vite BI 前端工程。后端在 `../be`，本地生产访问由 `be/cmd/bi-api` 托管 `/bi` 和 `/bi/*`。

## 常用命令

```bash
npm install
npm run build
npm run dev
```

从后端目录也可以执行：

```bash
cd ../be
make frontend-build
```

开发服务器默认监听 `127.0.0.1`，生产构建产物输出到 `dist/`。

## SDD

- `openspec/`: 轻量 OpenSpec，用来记录变更 proposal、design、tasks 和能力 spec。
- `specs/`: Spec Kit 风格的 feature spec、plan、tasks、quickstart。
- `.specify/`: Spec Kit 项目约束和长期约定。
