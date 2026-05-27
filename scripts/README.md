# Repository Scripts Guide

根目录 `scripts/` 只放跨前后端的仓库级脚本。

- `verify/verify_harness.sh`
  统一检查 Harness 文档、CI 入口、后端 Go 测试和前端构建。

后端运行、基础设施和链路验收脚本在 `be/scripts/`。日常优先用根目录 Makefile：

1. `make harness-check`
2. `make up`
3. `make status`
4. `make verify`
5. `make down`
