# AutoCert Makefile

.PHONY: build clean test install dist help

# 变量定义
BINARY_NAME=autocert
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GO_BUILD_FLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)"

# 目录变量
DIST_DIR=dist
BUILD_DIR=build

# 默认目标
help: ## 显示帮助信息
	@echo "AutoCert 构建系统"
	@echo ""
	@echo "可用目标:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## 构建当前平台的二进制文件
	@echo "构建 $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe,) .
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe,)"

build-linux: ## 构建 Linux 版本
	@GOOS=linux GOARCH=amd64 $(MAKE) build
	@echo "Linux 构建完成"

build-windows: ## 构建 Windows 版本
	@GOOS=windows GOARCH=amd64 $(MAKE) build
	@echo "Windows 构建完成"

build-darwin: ## 构建 macOS 版本
	@GOOS=darwin GOARCH=amd64 $(MAKE) build
	@echo "macOS 构建完成"

build-all: ## 构建所有平台版本
	@echo "构建所有平台版本..."
	@$(MAKE) build-linux
	@$(MAKE) build-windows  
	@$(MAKE) build-darwin
	@echo "所有平台构建完成"

test: ## 运行测试
	@echo "运行测试..."
	@go test -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

lint: ## 运行代码检查
	@echo "运行代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过检查"; \
		echo "安装方法: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...
	@echo "代码格式化完成"

clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "清理完成"

install: build ## 安装到系统
	@echo "安装 $(BINARY_NAME)..."
	@if [ "$(GOOS)" = "windows" ]; then \
		echo "Windows 系统请手动复制到 PATH 目录"; \
	else \
		sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/; \
		echo "安装完成: /usr/local/bin/$(BINARY_NAME)"; \
	fi

dist: ## 创建发布包
	@echo "创建发布包..."
	@mkdir -p $(DIST_DIR)
	
	# Linux amd64
	@GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_linux_amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	@rm $(DIST_DIR)/$(BINARY_NAME)
	
	# Linux arm64
	@GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_linux_arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	@rm $(DIST_DIR)/$(BINARY_NAME)
	
	# Windows amd64
	@GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME).exe
	@cd $(DIST_DIR) && zip $(BINARY_NAME)_$(VERSION)_windows_amd64.zip $(BINARY_NAME).exe
	@rm $(DIST_DIR)/$(BINARY_NAME).exe
	
	# Windows arm64
	@GOOS=windows GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME).exe
	@cd $(DIST_DIR) && zip $(BINARY_NAME)_$(VERSION)_windows_arm64.zip $(BINARY_NAME).exe
	@rm $(DIST_DIR)/$(BINARY_NAME).exe
	
	# macOS amd64
	@GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_darwin_amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	@rm $(DIST_DIR)/$(BINARY_NAME)
	
	# macOS arm64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_darwin_arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)
	@rm $(DIST_DIR)/$(BINARY_NAME)
	
	@echo "发布包创建完成:"
	@ls -la $(DIST_DIR)/

package: ## 一键打包所有平台（标准格式）
	@echo "一键打包 AutoCert 所有平台..."
	@mkdir -p $(DIST_DIR)
	@chmod +x scripts/package.sh
	@scripts/package.sh $(VERSION) $(DIST_DIR) $(BINARY_NAME)
	@echo "打包完成!"

package-linux: ## 打包 Linux 平台
	@echo "打包 Linux 平台..."
	@mkdir -p $(DIST_DIR)
	@chmod +x scripts/package.sh
	@scripts/package.sh $(VERSION) $(DIST_DIR) $(BINARY_NAME) linux

package-windows: ## 打包 Windows 平台
	@echo "打包 Windows 平台..."
	@mkdir -p $(DIST_DIR)
	@chmod +x scripts/package.sh
	@scripts/package.sh $(VERSION) $(DIST_DIR) $(BINARY_NAME) windows

release: ## 一键发布（清理+构建+打包）
	@echo "开始一键发布流程..."
	@$(MAKE) clean
	@$(MAKE) test
	@$(MAKE) package
	@echo "发布完成!"

quick-package: ## 快速打包（不清理不测试）
	@echo "快速打包..."
	@mkdir -p $(DIST_DIR)
	@chmod +x scripts/package.sh
	@scripts/package.sh $(VERSION) $(DIST_DIR) $(BINARY_NAME)

deps: ## 安装开发依赖
	@echo "安装开发依赖..."
	@go mod tidy
	@go mod download
	@echo "依赖安装完成"

dev: ## 开发模式运行
	@echo "开发模式运行..."
	@go run . --help

demo: build ## 运行演示
	@echo "运行演示..."
	@echo "1. 检查系统环境:"
	@$(BUILD_DIR)/$(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe,) --help
	@echo ""
	@echo "2. 查看版本:"
	@$(BUILD_DIR)/$(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe,) version || true
	@echo ""
	@echo "更多功能请参考文档"

# 版本相关
version: ## 显示版本信息
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)" 
	@echo "提交哈希: $(COMMIT_HASH)"

# Docker 相关（可选）
docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker build -t $(BINARY_NAME):latest .
	@echo "Docker 镜像构建完成"

docker-run: ## 运行 Docker 容器
	@echo "运行 Docker 容器..."
	@docker run --rm -it $(BINARY_NAME):latest --help