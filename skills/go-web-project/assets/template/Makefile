MODULE  := {{MODULE}}
SERVICE := {{SERVICE}}
BIN     := bin/$(SERVICE)
# 加载本地 .env（不存在则跳过）并导出给命令；直接 go run 不会读取 .env。
-include .env
export
# 迁移工具走固定版本，无需全局安装，也不污染主模块的依赖图
GOOSE   := go run github.com/pressly/goose/v3/cmd/goose@v3.29.0

# check 是提交前的唯一验收入口：gofmt + vet + test。
# lint / vuln 需要额外安装工具，不纳入 check。
.PHONY: help build run test test-race test-integration vet fmt fmt-check tidy check lint vuln migrate-up migrate-down db-up db-down

help:
	@echo "build run test test-race test-integration vet fmt fmt-check tidy check lint vuln migrate-up migrate-down db-up db-down"

build:
	go build -o $(BIN) ./cmd/server

run:
	go run ./cmd/server

test:
	go test -shuffle=on ./...

# 竞态检测：需要 cgo 与 C 工具链。CI 默认执行；Windows 本地需 MinGW。
test-race:
	CGO_ENABLED=1 go test -race -shuffle=on ./...

# 集成测试（标签 integration）：需要 make db-up 与 .env 中的 DATABASE_URL；不进 check。
test-integration:
	go test -tags=integration -shuffle=on ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "以下文件未格式化："; echo "$$out"; exit 1; fi

tidy:
	go mod tidy

check: fmt-check vet test

# 需要安装 golangci-lint
lint:
	golangci-lint run

# 需要安装 govulncheck
vuln:
	govulncheck ./...

migrate-up:
	$(GOOSE) -dir db/migrations mysql "$$DATABASE_URL" up

migrate-down:
	$(GOOSE) -dir db/migrations mysql "$$DATABASE_URL" down

db-up:
	docker compose up -d db

db-down:
	docker compose down
