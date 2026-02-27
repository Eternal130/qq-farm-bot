APP_NAME := qq-farm-bot
GOPROXY := https://goproxy.cn,direct

.PHONY: all clean frontend backend dev

all: frontend backend

# Build Vue frontend → outputs to cmd/server/dist/
frontend:
	@echo ">>> 构建前端..."
	cd web && npm run build
	@echo ">>> 前端构建完成"

# Build Go backend (embeds frontend)
backend:
	@echo ">>> 构建后端..."
	GOPROXY=$(GOPROXY) go build -o $(APP_NAME) ./cmd/server/
	@echo ">>> 构建完成: $(APP_NAME)"

# Install frontend dependencies
deps:
	@echo ">>> 安装前端依赖..."
	cd web && npm install
	@echo ">>> 安装Go依赖..."
	GOPROXY=$(GOPROXY) go mod tidy

# Dev mode: run frontend dev server + Go backend
dev:
	@echo "先启动Go后端: go run ./cmd/server/"
	@echo "再启动前端:   cd web && npm run dev"

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	rm -rf cmd/server/dist
	rm -rf web/dist
