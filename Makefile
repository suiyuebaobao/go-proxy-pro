.PHONY: build run clean test web all

# 仅构建后端
build:
	go build -o bin/server ./cmd/server

# 仅运行后端
run:
	go run ./cmd/server

# 清理
clean:
	rm -rf bin/
	rm -rf internal/handler/dist/*

# 测试
test:
	go test -v ./...

# 安装后端依赖
deps:
	go mod tidy

# 格式化
fmt:
	go fmt ./...

# 检查
lint:
	golangci-lint run

# 前端开发（需要后端同时运行）
web-dev:
	cd web && npm run dev

# 安装前端依赖
web-deps:
	cd web && npm install

# 构建前端
web-build:
	cd web && npm run build
	# 预压缩静态资源，避免在线 gzip 的 chunked 传输导致浏览器偶发 ERR_INCOMPLETE_CHUNKED_ENCODING
	# 只压缩 js/css（字体本身已压缩，没必要）
	find web/dist/assets -type f \( -name '*.js' -o -name '*.css' \) -exec gzip -kf -9 {} \;
	rm -rf internal/handler/dist/*
	cp -r web/dist/* internal/handler/dist/

# 完整构建（前端 + 后端）
all: web-build build
	@echo "Build complete: bin/server"

# 开发模式（热重载需要安装 air）
dev:
	air
