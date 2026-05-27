SHELL := /bin/bash

.DEFAULT_GOAL := help

.PHONY: help fmt test frontend-install frontend-build frontend-dev mac-start up down start stop status verify harness-check verify-debezium \
	infra-up infra-down debezium-up debezium-down clean

help:
	@printf "%-22s %s\n" "make up" "一键启动基础设施和 4 个服务"
	@printf "%-22s %s\n" "make down" "一键关闭 4 个服务和基础设施"
	@printf "%-22s %s\n" "make start" "仅启动 4 个服务"
	@printf "%-22s %s\n" "make stop" "仅停止 4 个服务"
	@printf "%-22s %s\n" "make status" "查看服务状态与最近日志"
	@printf "%-22s %s\n" "make test" "执行 go test ./..."
	@printf "%-22s %s\n" "make frontend-install" "安装 React BI 前端依赖"
	@printf "%-22s %s\n" "make frontend-build" "构建 React BI 前端"
	@printf "%-22s %s\n" "make frontend-dev" "启动 React BI 前端开发服务器"
	@printf "%-22s %s\n" "make mac-start" "Mac 新环境一键检查、安装、启动和验收"
	@printf "%-22s %s\n" "make verify" "执行本地主链路验收"
	@printf "%-22s %s\n" "make harness-check" "执行 Harness 统一静态验收"
	@printf "%-22s %s\n" "make verify-debezium" "执行 Debezium 链路验收"
	@printf "%-22s %s\n" "make infra-up" "启动 MySQL / ClickHouse / NATS"
	@printf "%-22s %s\n" "make infra-down" "关闭 MySQL / ClickHouse / NATS"
	@printf "%-22s %s\n" "make debezium-up" "启动 Debezium"
	@printf "%-22s %s\n" "make debezium-down" "关闭 Debezium"
	@printf "%-22s %s\n" "make fmt" "格式化 Go 代码"
	@printf "%-22s %s\n" "make clean" "清理 logs/run/tmp 运行产物"

fmt:
	@gofmt -w $$(find cmd internal -name '*.go' -type f)

test:
	@go test ./...

frontend-install:
	@npm --prefix ../fe install

frontend-build:
	@if [[ ! -d ../fe/node_modules ]]; then npm --prefix ../fe install; fi
	@npm --prefix ../fe run build

frontend-dev:
	@if [[ ! -d ../fe/node_modules ]]; then npm --prefix ../fe install; fi
	@npm --prefix ../fe run dev

mac-start:
	@./scripts/ops/mac_local_start.sh

infra-up:
	@./scripts/dev/dev_base_stack_up.sh

infra-down:
	@./scripts/dev/dev_base_stack_down.sh

start:
	@./scripts/ops/start.sh

stop:
	@./scripts/ops/stop.sh

status:
	@./scripts/ops/status.sh

verify:
	@./scripts/verify/verify_local_stack.sh

harness-check:
	@../scripts/verify/verify_harness.sh

debezium-up:
	@./scripts/dev/dev_debezium_up.sh

debezium-down:
	@./scripts/dev/dev_debezium_down.sh

verify-debezium:
	@./scripts/verify/verify_debezium_pipeline.sh

up:
	@./scripts/ops/up.sh

down:
	@./scripts/ops/down.sh

clean:
	@rm -rf logs run tmp
