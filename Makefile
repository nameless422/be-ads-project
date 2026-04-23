SHELL := /bin/bash

.DEFAULT_GOAL := help

.PHONY: help fmt test up down start stop status verify verify-debezium \
	infra-up infra-down debezium-up debezium-down clean

help:
	@printf "%-22s %s\n" "make up" "一键启动基础设施和 4 个服务"
	@printf "%-22s %s\n" "make down" "一键关闭 4 个服务和基础设施"
	@printf "%-22s %s\n" "make start" "仅启动 4 个服务"
	@printf "%-22s %s\n" "make stop" "仅停止 4 个服务"
	@printf "%-22s %s\n" "make status" "查看服务状态与最近日志"
	@printf "%-22s %s\n" "make test" "执行 go test ./..."
	@printf "%-22s %s\n" "make verify" "执行本地主链路验收"
	@printf "%-22s %s\n" "make verify-debezium" "执行 Debezium 链路验收"
	@printf "%-22s %s\n" "make infra-up" "启动 MySQL / ClickHouse / NATS"
	@printf "%-22s %s\n" "make infra-down" "关闭 MySQL / ClickHouse / NATS"
	@printf "%-22s %s\n" "make debezium-up" "启动 Debezium"
	@printf "%-22s %s\n" "make debezium-down" "关闭 Debezium"
	@printf "%-22s %s\n" "make fmt" "格式化 Go 代码"
	@printf "%-22s %s\n" "make clean" "清理 logs/ 和 run/ 下的运行产物"

fmt:
	@gofmt -w $$(find cmd internal -name '*.go' -type f)

test:
	@go test ./...

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
	@find logs run -mindepth 1 -maxdepth 1 -exec rm -rf {} +
