# AirLink Makefile

.PHONY: all build build-all dev clean run test

# 变量
BINARY_NAME=airlink
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# 构建目录
BUILD_DIR=dist

# 默认目标
all: build

# 开发运行
dev:
	@echo "启动开发服务器..."
	go run cmd/airlink/main.go --debug

# 本地构建
build:
	@echo "构建 $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/airlink/main.go

# 跨平台构建
build-all: clean
	@echo "构建所有平台..."
	@mkdir -p $(BUILD_DIR)

	@echo "构建 Windows (amd64)..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/airlink/main.go

	@echo "构建 macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/airlink/main.go

	@echo "构建 macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/airlink/main.go

	@echo "构建 Linux (amd64)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/airlink/main.go

	@echo "构建完成！"
	@ls -lh $(BUILD_DIR)

# 运行
run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

# 测试
test:
	go test -v -race -coverprofile=coverage.out ./...

# 清理
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

# 安装依赖
deps:
	go mod download
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 显示帮助
help:
	@echo "AirLink 构建工具"
	@echo ""
	@echo "使用方法："
	@echo "  make dev         - 启动开发服务器"
	@echo "  make build       - 构建当前平台"
	@echo "  make build-all   - 构建所有平台"
	@echo "  make run         - 构建并运行"
	@echo "  make test        - 运行测试"
	@echo "  make clean       - 清理构建文件"
	@echo "  make deps        - 安装依赖"
	@echo "  make fmt         - 格式化代码"
	@echo "  make lint        - 代码检查"
