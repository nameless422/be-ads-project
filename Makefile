SHELL := /bin/bash
PATH := /opt/homebrew/bin:/usr/local/go/bin:/usr/local/bin:$(PATH)

.DEFAULT_GOAL := help

BE_DIR := be
FE_DIR := fe
HARNESS_CHECK := ./scripts/verify/verify_harness.sh

.PHONY: help fmt test frontend-install frontend-build frontend-dev frontend-preview mac-start \
	up down start stop status verify harness-check verify-debezium \
	infra-up infra-down debezium-up debezium-down clean

help:
	@printf "%-22s %s\n" "make up" "一键启动后端基础设施和 4 个服务"
	@printf "%-22s %s\n" "make down" "一键关闭后端 4 个服务和基础设施"
	@printf "%-22s %s\n" "make start" "仅启动后端 4 个服务"
	@printf "%-22s %s\n" "make stop" "仅停止后端 4 个服务"
	@printf "%-22s %s\n" "make status" "查看后端服务状态与最近日志"
	@printf "%-22s %s\n" "make harness-check" "执行 Harness 统一静态验收"
	@printf "%-22s %s\n" "make test" "执行后端 go test ./..."
	@printf "%-22s %s\n" "make frontend-install" "安装前端依赖"
	@printf "%-22s %s\n" "make frontend-build" "构建 React/Vite 前端"
	@printf "%-22s %s\n" "make frontend-dev" "启动前端 Vite dev server"
	@printf "%-22s %s\n" "make frontend-preview" "预览前端构建产物"
	@printf "%-22s %s\n" "make mac-start" "Mac 新环境一键检查、安装、启动和验收"
	@printf "%-22s %s\n" "make verify" "执行后端本地主链路验收"
	@printf "%-22s %s\n" "make verify-debezium" "执行后端 Debezium 链路验收"
	@printf "%-22s %s\n" "make infra-up" "启动后端 MySQL / ClickHouse / NATS"
	@printf "%-22s %s\n" "make infra-down" "关闭后端 MySQL / ClickHouse / NATS"
	@printf "%-22s %s\n" "make debezium-up" "启动后端 Debezium"
	@printf "%-22s %s\n" "make debezium-down" "关闭后端 Debezium"
	@printf "%-22s %s\n" "make fmt" "格式化后端 Go 代码"
	@printf "%-22s %s\n" "make clean" "清理后端 be/logs 和 be/run 运行产物"

fmt:
	@$(MAKE) -C $(BE_DIR) fmt

test:
	@$(MAKE) -C $(BE_DIR) test

frontend-install:
	@cd $(FE_DIR) && npm ci

frontend-build:
	@cd $(FE_DIR) && npm run build

frontend-dev:
	@cd $(FE_DIR) && npm run dev

frontend-preview:
	@cd $(FE_DIR) && npm run preview

mac-start:
	@$(MAKE) -C $(BE_DIR) mac-start

infra-up:
	@$(MAKE) -C $(BE_DIR) infra-up

infra-down:
	@$(MAKE) -C $(BE_DIR) infra-down

start:
	@$(MAKE) -C $(BE_DIR) start

stop:
	@$(MAKE) -C $(BE_DIR) stop

status:
	@$(MAKE) -C $(BE_DIR) status

verify:
	@$(MAKE) -C $(BE_DIR) verify

harness-check:
	@$(HARNESS_CHECK)

debezium-up:
	@$(MAKE) -C $(BE_DIR) debezium-up

debezium-down:
	@$(MAKE) -C $(BE_DIR) debezium-down

verify-debezium:
	@$(MAKE) -C $(BE_DIR) verify-debezium

up:
	@$(MAKE) -C $(BE_DIR) up

down:
	@$(MAKE) -C $(BE_DIR) down

clean:
	@$(MAKE) -C $(BE_DIR) clean
